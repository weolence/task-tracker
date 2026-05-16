package httpadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"auth-service/internal/core/domain"
	"auth-service/internal/testhelper"

	"golang.org/x/crypto/bcrypt"
)

// buildAdminMux wires up a realistic admin mux (auth + adminOnly middleware)
// the same way main.go does, pointing proxy calls at projectServiceURL.
func buildAdminMux(repo *testhelper.MockUserRepository, projectServiceURL string) http.Handler {
	uc := newTestUC(repo)
	adminHandler := NewAdminHandler(uc, projectServiceURL)
	mux := http.NewServeMux()

	chain := func(h http.HandlerFunc) http.Handler {
		return AuthMiddleware(testSecretBytes)(AdminOnly(h))
	}

	mux.Handle("/admin/api/users/get", chain(adminHandler.GetUser))
	mux.Handle("/admin/api/users", chain(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			adminHandler.UpdateUser(w, r)
		case http.MethodDelete:
			adminHandler.DeleteUser(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	return mux
}

// ---- Access control (auth + admin gate) ---------------------------------

func TestAdminEndpoint_NoToken_Returns401(t *testing.T) {
	mux := buildAdminMux(testhelper.NewMockRepo(), "")
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(`{"user_id":1}`))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestAdminEndpoint_UserRoleToken_Returns403(t *testing.T) {
	mux := buildAdminMux(testhelper.NewMockRepo(), "")
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(`{"user_id":1}`))
	r.Header.Set("Authorization", "Bearer "+makeUserToken(1))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", w.Code)
	}
}

// Security: an expired admin token must not gain access.
func TestAdminEndpoint_ExpiredAdminToken_Returns401(t *testing.T) {
	expiredTok := makeToken(validClaims(1, domain.RoleAdmin, -time.Hour), testSecretBytes)
	mux := buildAdminMux(testhelper.NewMockRepo(), "")
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(`{"user_id":1}`))
	r.Header.Set("Authorization", "Bearer "+expiredTok)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

// Security: a token with an unrecognised role is now rejected by AuthMiddleware
// itself (401) before even reaching AdminOnly — the token is treated as invalid.
func TestAdminEndpoint_UnknownRoleToken_Returns401(t *testing.T) {
	forgeTok := makeToken(map[string]interface{}{
		"user_id": float64(1),
		"role":    "superadmin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, testSecretBytes)
	mux := buildAdminMux(testhelper.NewMockRepo(), "")
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(`{"user_id":1}`))
	r.Header.Set("Authorization", "Bearer "+forgeTok)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestAdminEndpoint_ValidAdminToken_Returns200(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "pass")
	stored := repo.GetStoredUser("a@b.com")

	mux := buildAdminMux(repo, "")
	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+makeAdminToken(1))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

// ---- GetUser ------------------------------------------------------------

func TestAdminGetUser_NeitherParam_Returns400(t *testing.T) {
	repo := testhelper.NewMockRepo()
	h := NewAdminHandler(newTestUC(repo), "")

	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.GetUser(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestAdminGetUser_ByID_NotFound_Returns404(t *testing.T) {
	h := NewAdminHandler(newTestUC(testhelper.NewMockRepo()), "")

	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(`{"user_id":9999}`))
	w := httptest.NewRecorder()
	h.GetUser(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}

func TestAdminGetUser_ByID_ReturnsUser(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "find@b.com", "pass")
	stored := repo.GetStoredUser("find@b.com")
	h := NewAdminHandler(newTestUC(repo), "")

	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.GetUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["email"] != "find@b.com" {
		t.Fatalf("expected email 'find@b.com', got %v", resp["email"])
	}
}

func TestAdminGetUser_ByEmail_ReturnsUser(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "byemail@b.com", "pass")
	h := NewAdminHandler(newTestUC(repo), "")

	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get",
		strings.NewReader(`{"email":"byemail@b.com"}`))
	w := httptest.NewRecorder()
	h.GetUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

// Security: admin user-lookup response must never include the password hash.
func TestAdminGetUser_ResponseContainsNoPassword(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "topsecret")
	stored := repo.GetStoredUser("a@b.com")
	h := NewAdminHandler(newTestUC(repo), "")

	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodGet, "/admin/api/users/get", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.GetUser(w, r)

	if strings.Contains(w.Body.String(), "topsecret") || strings.Contains(w.Body.String(), stored.Password) {
		t.Fatal("response must not contain plaintext or hashed password")
	}
}

// ---- UpdateUser ---------------------------------------------------------

func TestAdminUpdateUser_MissingRequiredFields_Returns400(t *testing.T) {
	h := NewAdminHandler(newTestUC(testhelper.NewMockRepo()), "")

	// user object missing email
	r := httptest.NewRequest(http.MethodPut, "/admin/api/users",
		strings.NewReader(`{"user":{"id":1,"name":"A","surname":"B","role":"user"}}`))
	w := httptest.NewRecorder()
	h.UpdateUser(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestAdminUpdateUser_InvalidRole_Returns400(t *testing.T) {
	h := NewAdminHandler(newTestUC(testhelper.NewMockRepo()), "")

	r := httptest.NewRequest(http.MethodPut, "/admin/api/users",
		strings.NewReader(`{"user":{"id":1,"email":"a@b.com","name":"A","surname":"B","role":"superadmin"}}`))
	w := httptest.NewRecorder()
	h.UpdateUser(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestAdminUpdateUser_Success_Returns200(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "before@b.com", "pass")
	stored := repo.GetStoredUser("before@b.com")
	h := NewAdminHandler(newTestUC(repo), "")

	body := fmt.Sprintf(`{"user":{"id":%d,"email":"after@b.com","name":"New","surname":"Name","role":"user"}}`, stored.ID)
	r := httptest.NewRequest(http.MethodPut, "/admin/api/users", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200 (body: %s)", w.Code, w.Body.String())
	}
}

// When a new password is provided, it must be stored as a bcrypt hash, not plaintext.
func TestAdminUpdateUser_NewPasswordIsHashed(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "oldpass")
	stored := repo.GetStoredUser("a@b.com")
	h := NewAdminHandler(newTestUC(repo), "")

	const newPlain = "newplainpassword"
	body := fmt.Sprintf(`{"user":{"id":%d,"email":"a@b.com","name":"Test","surname":"User","role":"user"},"password":"%s"}`,
		stored.ID, newPlain)
	r := httptest.NewRequest(http.MethodPut, "/admin/api/users", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	after := repo.GetStoredUser("a@b.com")
	if after.Password == newPlain {
		t.Fatal("new password stored as plaintext – must be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(after.Password), []byte(newPlain)); err != nil {
		t.Fatalf("stored value is not a valid bcrypt hash of the new password: %v", err)
	}
}

// ---- DeleteUser ---------------------------------------------------------

func TestAdminDeleteUser_Success_Returns200(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "gone@b.com", "pass")
	stored := repo.GetStoredUser("gone@b.com")
	h := NewAdminHandler(newTestUC(repo), "")

	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodDelete, "/admin/api/users", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.DeleteUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	if repo.GetStoredUser("gone@b.com") != nil {
		t.Fatal("user still exists after deletion")
	}
}

// ---- Proxy (project service forwarding) ---------------------------------

// The proxy must forward the Authorization header to the downstream service.
func TestAdminProxy_ForwardsAuthorizationHeader(t *testing.T) {
	var gotAuth string
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer downstream.Close()

	repo := testhelper.NewMockRepo()
	h := NewAdminHandler(newTestUC(repo), downstream.URL)

	adminTok := makeAdminToken(1)
	r := httptest.NewRequest(http.MethodGet, "/admin/api/projects/get", strings.NewReader(`{}`))
	r.Header.Set("Authorization", "Bearer "+adminTok)
	w := httptest.NewRecorder()
	h.ProxyProject(w, r, "/api/admin/projects/get")

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	want := "Bearer " + adminTok
	if gotAuth != want {
		t.Fatalf("Authorization not forwarded: got %q, want %q", gotAuth, want)
	}
}

// If the project service is down, the proxy must return 502, not panic or hang.
func TestAdminProxy_ServiceUnavailable_Returns502(t *testing.T) {
	h := NewAdminHandler(newTestUC(testhelper.NewMockRepo()), "http://127.0.0.1:19999")

	r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ProxyProject(w, r, "/api/admin/projects/get")

	if w.Code != http.StatusBadGateway {
		t.Fatalf("got %d, want 502", w.Code)
	}
}

// ---- Role promotion via UpdateUser (privilege escalation check) ---------

// An admin can promote a user to admin via UpdateUser.
// Verify the role is stored correctly and is the only way to escalate.
func TestAdminUpdateUser_RolePromotion_StoredCorrectly(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "plain@b.com", "pass")
	stored := repo.GetStoredUser("plain@b.com")

	if stored.Role != domain.RoleUser {
		t.Fatalf("precondition: expected role 'user', got %q", stored.Role)
	}

	h := NewAdminHandler(newTestUC(repo), "")
	body := fmt.Sprintf(`{"user":{"id":%d,"email":"plain@b.com","name":"Test","surname":"User","role":"admin"}}`, stored.ID)
	r := httptest.NewRequest(http.MethodPut, "/admin/api/users", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.UpdateUser(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("UpdateUser failed: %d %s", w.Code, w.Body.String())
	}

	// Obtain a fresh token and validate that the role change is reflected.
	uc := newTestUC(repo)
	tok, err := uc.Login(context.Background(), "plain@b.com", "pass")
	if err != nil {
		t.Fatalf("login after promotion failed: %v", err)
	}
	payload, err := uc.ValidateToken(context.Background(), tok)
	if err != nil {
		t.Fatalf("token validation failed: %v", err)
	}
	if payload.Role != domain.RoleAdmin {
		t.Fatalf("role not persisted: got %q, want %q", payload.Role, domain.RoleAdmin)
	}
}

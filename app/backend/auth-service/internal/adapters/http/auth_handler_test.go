package httpadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"auth-service/internal/testhelper"
)

// buildAuthMux mirrors the main.go route setup for the auth handler,
// including the AuthMiddleware guard on /user-info.
func buildAuthMux(repo *testhelper.MockUserRepository) http.Handler {
	uc := newTestUC(repo)
	h := NewAuthHandler(uc)
	mux := http.NewServeMux()
	mux.HandleFunc("/login", h.Login)
	mux.HandleFunc("/validate-token", h.ValidateToken)
	mux.Handle("/user-info", AuthMiddleware(testSecretBytes)(http.HandlerFunc(h.GetUserInfo)))
	return mux
}

// ---- /user-info route-level auth (integration) --------------------------

func TestUserInfo_Unauthenticated_Returns401(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "victim@b.com", "pass")
	stored := repo.GetStoredUser("victim@b.com")
	mux := buildAuthMux(repo)

	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodPost, "/user-info", strings.NewReader(body))
	// No Authorization header – simulates an unauthenticated caller
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /user-info must return 401, got %d", w.Code)
	}
}

func TestUserInfo_WithValidToken_Returns200(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "pass")
	stored := repo.GetStoredUser("a@b.com")
	mux := buildAuthMux(repo)

	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodPost, "/user-info", strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+makeUserToken(int(stored.ID)))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("authenticated /user-info must return 200, got %d", w.Code)
	}
}

// ---- Login --------------------------------------------------------------

func TestLogin_MalformedBody_Returns400(t *testing.T) {
	repo := testhelper.NewMockRepo()
	h := NewAuthHandler(newTestUC(repo))

	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("not json"))
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestLogin_InvalidCredentials_Returns401(t *testing.T) {
	repo := testhelper.NewMockRepo()
	h := NewAuthHandler(newTestUC(repo))

	body := `{"email":"ghost@example.com","password":"wrong"}`
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestLogin_WrongPassword_Returns401(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "correct")
	h := NewAuthHandler(newTestUC(repo))

	body := `{"email":"a@b.com","password":"wrong"}`
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestLogin_Success_Returns200WithToken(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "pass")
	h := NewAuthHandler(newTestUC(repo))

	body := `{"email":"a@b.com","password":"pass"}`
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.Login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	tok, ok := resp["token"].(string)
	if !ok || tok == "" {
		t.Fatalf("expected non-empty token in response, got: %v", resp)
	}
}

// Security: both "user not found" and "wrong password" must return 401,
// so HTTP callers cannot enumerate registered emails.
func TestLogin_UserEnumeration_BothReturn401(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "exists@b.com", "correct")
	h := NewAuthHandler(newTestUC(repo))

	notFoundReq := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(`{"email":"ghost@b.com","password":"pass"}`))
	w1 := httptest.NewRecorder()
	h.Login(w1, notFoundReq)

	wrongPassReq := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(`{"email":"exists@b.com","password":"wrong"}`))
	w2 := httptest.NewRecorder()
	h.Login(w2, wrongPassReq)

	if w1.Code != http.StatusUnauthorized || w2.Code != http.StatusUnauthorized {
		t.Fatalf("both cases must return 401; got not-found=%d, wrong-pass=%d", w1.Code, w2.Code)
	}
}

// ---- ValidateToken ------------------------------------------------------

func TestValidateToken_MalformedBody_Returns400(t *testing.T) {
	h := NewAuthHandler(newTestUC(testhelper.NewMockRepo()))

	r := httptest.NewRequest(http.MethodPost, "/validate-token", strings.NewReader("{bad json"))
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestValidateToken_InvalidToken_Returns401(t *testing.T) {
	h := NewAuthHandler(newTestUC(testhelper.NewMockRepo()))

	r := httptest.NewRequest(http.MethodPost, "/validate-token",
		strings.NewReader(`{"token":"garbage.token.value"}`))
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestValidateToken_Success_ReturnsUserIDAndRole(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "pass")
	uc := newTestUC(repo)
	h := NewAuthHandler(uc)

	// Obtain a token via the use case directly
	tok, _ := uc.Login(context.Background(), "a@b.com", "pass")

	body := fmt.Sprintf(`{"token":"%s"}`, tok)
	r := httptest.NewRequest(http.MethodPost, "/validate-token", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["role"] != "user" {
		t.Fatalf("expected role 'user', got: %v", resp["role"])
	}
	userID, ok := resp["user_id"].(float64)
	if !ok || userID <= 0 {
		t.Fatalf("expected positive user_id in response, got: %v", resp["user_id"])
	}
}

// Security: the response must never include a password field.
func TestValidateToken_ResponseContainsNoPassword(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "supersecret")
	uc := newTestUC(repo)
	h := NewAuthHandler(uc)
	tok, _ := uc.Login(context.Background(), "a@b.com", "supersecret")

	r := httptest.NewRequest(http.MethodPost, "/validate-token",
		strings.NewReader(fmt.Sprintf(`{"token":"%s"}`, tok)))
	w := httptest.NewRecorder()
	h.ValidateToken(w, r)

	if strings.Contains(w.Body.String(), "supersecret") {
		t.Fatal("response body must not contain the plaintext password")
	}
	if strings.Contains(w.Body.String(), "password") {
		t.Fatal("response body must not contain a 'password' field")
	}
}

// ---- GetUserInfo --------------------------------------------------------

func TestGetUserInfo_NeitherParam_Returns400(t *testing.T) {
	h := NewAuthHandler(newTestUC(testhelper.NewMockRepo()))

	r := httptest.NewRequest(http.MethodPost, "/user-info", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.GetUserInfo(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}

func TestGetUserInfo_ByID_NotFound_Returns404(t *testing.T) {
	h := NewAuthHandler(newTestUC(testhelper.NewMockRepo()))

	r := httptest.NewRequest(http.MethodPost, "/user-info",
		strings.NewReader(`{"user_id":9999}`))
	w := httptest.NewRecorder()
	h.GetUserInfo(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}

func TestGetUserInfo_ByID_ReturnsUser(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "pass")
	stored := repo.GetStoredUser("a@b.com")
	h := NewAuthHandler(newTestUC(repo))

	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodPost, "/user-info", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.GetUserInfo(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["email"] != "a@b.com" {
		t.Fatalf("expected email 'a@b.com', got %v", resp["email"])
	}
}

func TestGetUserInfo_ByEmail_ReturnsUser(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "named@b.com", "pass")
	h := NewAuthHandler(newTestUC(repo))

	r := httptest.NewRequest(http.MethodPost, "/user-info",
		strings.NewReader(`{"email":"named@b.com"}`))
	w := httptest.NewRecorder()
	h.GetUserInfo(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

// Security: the response must never expose the hashed password field.
func TestGetUserInfo_ResponseContainsNoPassword(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "a@b.com", "topsecretpassword")
	stored := repo.GetStoredUser("a@b.com")
	h := NewAuthHandler(newTestUC(repo))

	body := fmt.Sprintf(`{"user_id":%d}`, stored.ID)
	r := httptest.NewRequest(http.MethodPost, "/user-info", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.GetUserInfo(w, r)

	if strings.Contains(w.Body.String(), "topsecretpassword") {
		t.Fatal("plaintext password must not appear in response")
	}
	if strings.Contains(w.Body.String(), stored.Password) {
		t.Fatal("bcrypt hash must not appear in response")
	}
}


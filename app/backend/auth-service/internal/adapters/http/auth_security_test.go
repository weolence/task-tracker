package httpadapter

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userv1 "auth-service/api/proto/userv1"
	"auth-service/internal/testhelper"

	"google.golang.org/protobuf/encoding/protojson"
)

func newSecurityHandler(repo *testhelper.MockUserRepository) *AuthHandler {
	return NewAuthHandler(newTestUC(repo))
}

func TestSecurityLogin_CorrectCredentials_Returns200(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "alice@example.com", "s3cret!")
	handler := newSecurityHandler(repo)

	body, _ := protojson.Marshal(&userv1.LoginRequest{Email: "alice@example.com", Password: "s3cret!"})
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.Login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp userv1.LoginResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestSecurityLogin_WrongPassword_Returns401(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "bob@example.com", "correctpass")
	handler := newSecurityHandler(repo)

	body, _ := protojson.Marshal(&userv1.LoginRequest{Email: "bob@example.com", Password: "wrongpass"})
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSecurityLogin_NonExistentUser_Returns401(t *testing.T) {
	repo := testhelper.NewMockRepo()
	handler := newSecurityHandler(repo)

	body, _ := protojson.Marshal(&userv1.LoginRequest{Email: "ghost@example.com", Password: "pass"})
	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSecurityValidateToken_ValidToken_Returns200(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "carol@example.com", "pass")
	handler := newSecurityHandler(repo)

	// login to get token
	loginBody, _ := protojson.Marshal(&userv1.LoginRequest{Email: "carol@example.com", Password: "pass"})
	lr := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(string(loginBody)))
	lw := httptest.NewRecorder()
	handler.Login(lw, lr)

	var loginResp userv1.LoginResponse
	json.Unmarshal(lw.Body.Bytes(), &loginResp)

	// validate
	valBody, _ := protojson.Marshal(&userv1.ValidateTokenRequest{Token: loginResp.Token})
	r := httptest.NewRequest(http.MethodPost, "/validate-token", strings.NewReader(string(valBody)))
	w := httptest.NewRecorder()

	handler.ValidateToken(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestSecurityValidateToken_InvalidToken_Returns401(t *testing.T) {
	repo := testhelper.NewMockRepo()
	handler := newSecurityHandler(repo)

	body, _ := protojson.Marshal(&userv1.ValidateTokenRequest{Token: "not.a.valid.jwt"})
	r := httptest.NewRequest(http.MethodPost, "/validate-token", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.ValidateToken(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSecurityGetUserInfo_ExistingUser_Returns200(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "dave@example.com", "pass")
	handler := newSecurityHandler(repo)

	userID := int32(1) // first user seeded gets ID 1
	body, _ := protojson.Marshal(&userv1.GetUserRequest{UserId: &userID})
	r := httptest.NewRequest(http.MethodPost, "/user-info", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.GetUserInfo(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestSecurityGetUserInfo_NonExistentUser_Returns404(t *testing.T) {
	repo := testhelper.NewMockRepo()
	handler := newSecurityHandler(repo)

	userID := int32(999)
	body, _ := protojson.Marshal(&userv1.GetUserRequest{UserId: &userID})
	r := httptest.NewRequest(http.MethodPost, "/user-info", strings.NewReader(string(body)))
	w := httptest.NewRecorder()

	handler.GetUserInfo(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestSecurityPassword_NotStoredInPlaintext(t *testing.T) {
	repo := testhelper.NewMockRepo()
	seedUser(repo, "eve@example.com", "mysecretpassword")

	stored := repo.GetStoredUser("eve@example.com")
	if stored == nil {
		t.Fatal("user not found in repo")
	}
	if stored.Password == "mysecretpassword" {
		t.Error("password must not be stored in plaintext")
	}
	if stored.Password == "" {
		t.Error("password field must not be empty")
	}
}

func TestSecurityLogin_EmptyBody_Returns400(t *testing.T) {
	repo := testhelper.NewMockRepo()
	handler := newSecurityHandler(repo)

	r := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(""))
	w := httptest.NewRecorder()

	handler.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

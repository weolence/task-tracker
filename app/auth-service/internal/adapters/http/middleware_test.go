package httpadapter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"auth-service/internal/core/domain"
)

// ---- extractBearerToken -------------------------------------------------

func TestExtractBearerToken_NoHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := extractBearerToken(r)
	if err == nil {
		t.Fatal("expected error for missing Authorization header")
	}
}

func TestExtractBearerToken_EmptyHeader(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "")
	_, err := extractBearerToken(r)
	if err == nil {
		t.Fatal("expected error for empty Authorization header")
	}
}

func TestExtractBearerToken_MissingBearerPrefix(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "sometoken")
	_, err := extractBearerToken(r)
	if err == nil {
		t.Fatal("expected error when 'Bearer' prefix is absent")
	}
}

// The prefix check is case-sensitive: "bearer" must not be accepted.
func TestExtractBearerToken_LowercasePrefixRejected(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "bearer sometoken")
	_, err := extractBearerToken(r)
	if err == nil {
		t.Fatal("lowercase 'bearer' prefix must be rejected")
	}
}

func TestExtractBearerToken_BearerWithNoToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer ")
	_, err := extractBearerToken(r)
	if err == nil {
		t.Fatal("expected error for 'Bearer ' with no token")
	}
}

func TestExtractBearerToken_Valid(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer mytoken123")
	tok, err := extractBearerToken(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "mytoken123" {
		t.Fatalf("got %q, want %q", tok, "mytoken123")
	}
}

// ---- parseTokenClaims ---------------------------------------------------

func TestParseTokenClaims_Gibberish(t *testing.T) {
	_, _, err := parseTokenClaims("not.valid.jwt", testSecretBytes)
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}

func TestParseTokenClaims_WrongSecret(t *testing.T) {
	tok := makeToken(validClaims(1, domain.RoleUser, time.Hour), []byte("other-secret-key-totally-diff!!"))
	_, _, err := parseTokenClaims(tok, testSecretBytes)
	if err == nil {
		t.Fatal("expected error for token signed with wrong secret")
	}
}

func TestParseTokenClaims_ExpiredToken(t *testing.T) {
	tok := makeToken(validClaims(1, domain.RoleUser, -time.Hour), testSecretBytes)
	_, _, err := parseTokenClaims(tok, testSecretBytes)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

// Security: alg=none attack at the middleware parsing layer.
func TestParseTokenClaims_NoneAlgorithmRejected(t *testing.T) {
	noneToken := rawJWT("none", map[string]interface{}{
		"user_id": 1, "role": "admin",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, "")
	_, _, err := parseTokenClaims(noneToken, testSecretBytes)
	if err == nil {
		t.Fatal("alg=none token must be rejected by middleware")
	}
}

// Security: algorithm confusion – RS256 token must be rejected when server expects HS256.
func TestParseTokenClaims_RSAlgorithmRejected(t *testing.T) {
	rsToken := rawJWT("RS256", map[string]interface{}{
		"user_id": 1, "role": "admin",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, "fakesig")
	_, _, err := parseTokenClaims(rsToken, testSecretBytes)
	if err == nil {
		t.Fatal("RS256 token must be rejected when server expects HS256")
	}
}

// After the fix, tokens with a missing or unrecognised role must be rejected
// at the middleware layer, not silently defaulted to "user".
func TestParseTokenClaims_MissingRole_Rejected(t *testing.T) {
	tok := makeToken(map[string]interface{}{
		"user_id": float64(1),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, testSecretBytes)
	_, _, err := parseTokenClaims(tok, testSecretBytes)
	if err == nil {
		t.Fatal("token without role claim must be rejected")
	}
}

func TestParseTokenClaims_InvalidRole_Rejected(t *testing.T) {
	tok := makeToken(map[string]interface{}{
		"user_id": float64(1),
		"role":    "superadmin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}, testSecretBytes)
	_, _, err := parseTokenClaims(tok, testSecretBytes)
	if err == nil {
		t.Fatal("token with unrecognised role must be rejected by middleware")
	}
}

func TestParseTokenClaims_MissingUserID(t *testing.T) {
	tok := makeToken(map[string]interface{}{
		"role": domain.RoleUser,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}, testSecretBytes)
	_, _, err := parseTokenClaims(tok, testSecretBytes)
	if err == nil {
		t.Fatal("expected error for token without user_id claim")
	}
}

func TestParseTokenClaims_ValidUserToken(t *testing.T) {
	tok := makeToken(validClaims(42, domain.RoleUser, time.Hour), testSecretBytes)
	userID, role, err := parseTokenClaims(tok, testSecretBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != 42 {
		t.Fatalf("got user_id %d, want 42", userID)
	}
	if role != domain.RoleUser {
		t.Fatalf("got role %q, want %q", role, domain.RoleUser)
	}
}

func TestParseTokenClaims_ValidAdminToken(t *testing.T) {
	tok := makeToken(validClaims(7, domain.RoleAdmin, time.Hour), testSecretBytes)
	userID, role, err := parseTokenClaims(tok, testSecretBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != 7 {
		t.Fatalf("got user_id %d, want 7", userID)
	}
	if role != domain.RoleAdmin {
		t.Fatalf("got role %q, want %q", role, domain.RoleAdmin)
	}
}

// ---- AuthMiddleware (full HTTP handler) ---------------------------------

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthMiddleware_NoToken_Returns401(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	AuthMiddleware(testSecretBytes)(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken_Returns401(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer definitely-not-a-jwt")
	w := httptest.NewRecorder()
	AuthMiddleware(testSecretBytes)(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	tok := makeToken(validClaims(1, domain.RoleUser, -time.Hour), testSecretBytes)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	AuthMiddleware(testSecretBytes)(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_NoneAlgToken_Returns401(t *testing.T) {
	noneToken := rawJWT("none", map[string]interface{}{
		"user_id": 1, "role": "admin",
		"exp": time.Now().Add(time.Hour).Unix(),
	}, "")
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+noneToken)
	w := httptest.NewRecorder()
	AuthMiddleware(testSecretBytes)(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("alg=none token passed middleware: got %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_WrongSecretToken_Returns401(t *testing.T) {
	tok := makeToken(validClaims(1, domain.RoleUser, time.Hour), []byte("wrong-secret-key-totally-diff!!!"))
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	AuthMiddleware(testSecretBytes)(okHandler()).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d, want 401", w.Code)
	}
}

// A valid token must pass the middleware and inject user_id and role into context.
func TestAuthMiddleware_ValidToken_InjectsContext(t *testing.T) {
	tok := makeToken(validClaims(99, domain.RoleUser, time.Hour), testSecretBytes)

	var gotID, gotRole interface{}
	capture := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotID = r.Context().Value(userIDKey)
		gotRole = r.Context().Value(userRoleKey)
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	AuthMiddleware(testSecretBytes)(capture).ServeHTTP(w, r)

	if gotID != int32(99) {
		t.Fatalf("context user_id: got %v (%T), want 99 (int32)", gotID, gotID)
	}
	if gotRole != domain.RoleUser {
		t.Fatalf("context user_role: got %v, want %q", gotRole, domain.RoleUser)
	}
}

// ---- AdminOnly ----------------------------------------------------------

func adminGate() http.Handler {
	return AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestAdminOnly_NoContextValue_Returns403(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	adminGate().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", w.Code)
	}
}

func TestAdminOnly_UserRole_Returns403(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), userRoleKey, domain.RoleUser))
	w := httptest.NewRecorder()
	adminGate().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d, want 403", w.Code)
	}
}

// Security: any role other than the exact string "admin" must be forbidden.
// Note: invalid roles ("superadmin", "ADMIN", etc.) are now rejected at the
// AuthMiddleware layer (401) before reaching AdminOnly. This test covers the
// AdminOnly check directly, which still applies for the empty-string edge case
// and future callers that bypass the middleware in tests.
func TestAdminOnly_UnknownRole_Returns403(t *testing.T) {
	for _, role := range []string{"ADMIN", "Admin", "administrator", ""} {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(context.WithValue(r.Context(), userRoleKey, role))
		w := httptest.NewRecorder()
		adminGate().ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("role %q was not blocked: got %d, want 403", role, w.Code)
		}
	}
}

func TestAdminOnly_AdminRole_Returns200(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r = r.WithContext(context.WithValue(r.Context(), userRoleKey, domain.RoleAdmin))
	w := httptest.NewRecorder()
	adminGate().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", w.Code)
	}
}

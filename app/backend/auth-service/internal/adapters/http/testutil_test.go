package httpadapter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"auth-service/internal/core/domain"
	"auth-service/internal/core/usecase"
	"auth-service/internal/testhelper"

	"github.com/golang-jwt/jwt/v5"
)

// testSecretBytes is the shared HMAC secret used across all HTTP adapter tests.
var testSecretBytes = []byte("test-hmac-secret-for-http-adapter")

func makeToken(claims jwt.MapClaims, secret []byte) string {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	return tok
}

func validClaims(userID int, role string, dur time.Duration) jwt.MapClaims {
	return jwt.MapClaims{
		"user_id": float64(userID),
		"role":    role,
		"exp":     time.Now().Add(dur).Unix(),
	}
}

// rawJWT manually builds a JWT string with an arbitrary algorithm header.
// Used to craft algorithm-confusion attack tokens.
func rawJWT(alg string, claims map[string]interface{}, sig string) string {
	h, _ := json.Marshal(map[string]string{"alg": alg, "typ": "JWT"})
	p, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(h) + "." +
		base64.RawURLEncoding.EncodeToString(p) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(sig))
}

func makeUserToken(userID int) string {
	return makeToken(validClaims(userID, domain.RoleUser, time.Hour), testSecretBytes)
}

func makeAdminToken(userID int) string {
	return makeToken(validClaims(userID, domain.RoleAdmin, time.Hour), testSecretBytes)
}

func newTestUC(repo *testhelper.MockUserRepository) *usecase.AuthUseCase {
	return usecase.NewAuthUseCase(repo, testSecretBytes)
}

// seedUser registers a user in the mock repo via the use case.
func seedUser(repo *testhelper.MockUserRepository, email, password string) {
	uc := newTestUC(repo)
	_ = uc.Register(context.Background(), domain.User{
		Email: email, Password: password, Name: "Test", Surname: "User",
	})
}

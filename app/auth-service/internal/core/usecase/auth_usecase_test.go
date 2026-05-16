package usecase_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"auth-service/internal/core/domain"
	"auth-service/internal/core/usecase"
	"auth-service/internal/testhelper"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	testSecret = []byte("test-hmac-secret-for-usecase-test")
	bg         = context.Background()
)

// ---- helpers ------------------------------------------------------------

func newUC(repo *testhelper.MockUserRepository) *usecase.AuthUseCase {
	return usecase.NewAuthUseCase(repo, testSecret)
}

func mustRegister(t *testing.T, uc *usecase.AuthUseCase, email, password string) {
	t.Helper()
	err := uc.Register(bg, domain.User{Email: email, Password: password, Name: "Test", Surname: "User"})
	if err != nil {
		t.Fatalf("mustRegister: %v", err)
	}
}

func mustLogin(t *testing.T, uc *usecase.AuthUseCase, email, password string) string {
	t.Helper()
	tok, err := uc.Login(bg, email, password)
	if err != nil {
		t.Fatalf("mustLogin: %v", err)
	}
	return tok
}

// buildRawJWT manually constructs a JWT string with an arbitrary algorithm header.
// Used to craft algorithm-confusion attack tokens in tests.
func buildRawJWT(alg string, claims map[string]interface{}, sig string) string {
	h, _ := json.Marshal(map[string]string{"alg": alg, "typ": "JWT"})
	p, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(h) + "." +
		base64.RawURLEncoding.EncodeToString(p) + "." +
		base64.RawURLEncoding.EncodeToString([]byte(sig))
}

// ---- Register -----------------------------------------------------------

func TestRegister_EmptyEmail(t *testing.T) {
	err := newUC(testhelper.NewMockRepo()).Register(bg, domain.User{
		Email: "", Password: "pass", Name: "A", Surname: "B",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestRegister_EmptyPassword(t *testing.T) {
	err := newUC(testhelper.NewMockRepo()).Register(bg, domain.User{
		Email: "a@b.com", Password: "", Name: "A", Surname: "B",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestRegister_EmptyName(t *testing.T) {
	err := newUC(testhelper.NewMockRepo()).Register(bg, domain.User{
		Email: "a@b.com", Password: "pass", Name: "", Surname: "B",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestRegister_EmptySurname(t *testing.T) {
	err := newUC(testhelper.NewMockRepo()).Register(bg, domain.User{
		Email: "a@b.com", Password: "pass", Name: "A", Surname: "",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := testhelper.NewMockRepo()
	uc := newUC(repo)
	u := domain.User{Email: "dup@b.com", Password: "pass", Name: "A", Surname: "B"}
	_ = uc.Register(bg, u)
	err := uc.Register(bg, u)
	if !errors.Is(err, domain.ErrUserAlreadyExists) {
		t.Fatalf("got %v, want ErrUserAlreadyExists", err)
	}
}

func TestRegister_DBError(t *testing.T) {
	repo := testhelper.NewMockRepo()
	repo.GetByEmailErr = errors.New("connection refused")
	err := newUC(repo).Register(bg, domain.User{
		Email: "a@b.com", Password: "pass", Name: "A", Surname: "B",
	})
	if !errors.Is(err, domain.ErrConnectionToDB) {
		t.Fatalf("got %v, want ErrConnectionToDB", err)
	}
}

// Password must be stored as a bcrypt hash, never plaintext.
func TestRegister_PasswordNotStoredInPlaintext(t *testing.T) {
	repo := testhelper.NewMockRepo()
	const plain = "supersecret123"
	mustRegister(t, newUC(repo), "a@b.com", plain)

	stored := repo.GetStoredUser("a@b.com")
	if stored.Password == plain {
		t.Fatal("password stored in plaintext – must be hashed")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte(plain)); err != nil {
		t.Fatalf("stored value is not a valid bcrypt hash of the password: %v", err)
	}
}

// New users must always be assigned role "user", never "admin", by default.
func TestRegister_DefaultRoleIsUser(t *testing.T) {
	repo := testhelper.NewMockRepo()
	mustRegister(t, newUC(repo), "a@b.com", "pass")
	stored := repo.GetStoredUser("a@b.com")
	if stored.Role != domain.RoleUser {
		t.Fatalf("got role %q, want %q", stored.Role, domain.RoleUser)
	}
}

// ---- Login --------------------------------------------------------------

func TestLogin_EmptyEmail(t *testing.T) {
	_, err := newUC(testhelper.NewMockRepo()).Login(bg, "", "pass")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_EmptyPassword(t *testing.T) {
	_, err := newUC(testhelper.NewMockRepo()).Login(bg, "a@b.com", "")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_UnknownUser(t *testing.T) {
	_, err := newUC(testhelper.NewMockRepo()).Login(bg, "ghost@b.com", "pass")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := testhelper.NewMockRepo()
	uc := newUC(repo)
	mustRegister(t, uc, "a@b.com", "correct")
	_, err := uc.Login(bg, "a@b.com", "wrong")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestLogin_DBError(t *testing.T) {
	repo := testhelper.NewMockRepo()
	repo.GetByEmailErr = errors.New("db down")
	_, err := newUC(repo).Login(bg, "a@b.com", "pass")
	if !errors.Is(err, domain.ErrConnectionToDB) {
		t.Fatalf("got %v, want ErrConnectionToDB", err)
	}
}

func TestLogin_Success_ReturnsNonEmptyToken(t *testing.T) {
	repo := testhelper.NewMockRepo()
	uc := newUC(repo)
	mustRegister(t, uc, "a@b.com", "pass")
	tok, err := uc.Login(bg, "a@b.com", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok == "" {
		t.Fatal("expected non-empty token")
	}
}

// Security: "user not found" and "wrong password" must return the same error
// to prevent attackers from enumerating which emails are registered.
func TestLogin_UserEnumeration_SameErrorForBothCases(t *testing.T) {
	repo := testhelper.NewMockRepo()
	uc := newUC(repo)
	mustRegister(t, uc, "exists@b.com", "correct")

	_, errNoUser := uc.Login(bg, "ghost@b.com", "pass")
	_, errWrongPw := uc.Login(bg, "exists@b.com", "wrong")

	if errNoUser != errWrongPw {
		t.Fatalf("user existence leaked: no-user=%v, wrong-pw=%v", errNoUser, errWrongPw)
	}
}

// ---- ValidateToken ------------------------------------------------------

func TestValidateToken_EmptyString(t *testing.T) {
	_, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, "")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestValidateToken_Gibberish(t *testing.T) {
	_, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, "not.a.token")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

// Security: modifying even one character of the signature must invalidate the token.
func TestValidateToken_TamperedSignature(t *testing.T) {
	repo := testhelper.NewMockRepo()
	uc := newUC(repo)
	mustRegister(t, uc, "a@b.com", "pass")
	tok := mustLogin(t, uc, "a@b.com", "pass")

	tampered := tok[:len(tok)-4] + "XXXX"
	_, err := uc.ValidateToken(bg, tampered)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("tampered token was accepted: got %v, want ErrInvalidCredentials", err)
	}
}

// Security: a token signed with a different key must be rejected.
func TestValidateToken_WrongSecret(t *testing.T) {
	repo := testhelper.NewMockRepo()
	attackerUC := usecase.NewAuthUseCase(repo, []byte("attacker-secret-key-totally-diff"))
	_ = attackerUC.Register(bg, domain.User{Email: "x@b.com", Password: "p", Name: "X", Surname: "Y"})
	tok := mustLogin(t, attackerUC, "x@b.com", "p")

	_, err := newUC(repo).ValidateToken(bg, tok)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("token signed with wrong key was accepted: got %v, want ErrInvalidCredentials", err)
	}
}

// Security: expired tokens must be rejected even if signature is valid.
func TestValidateToken_ExpiredToken(t *testing.T) {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"role":    domain.RoleUser,
		"exp":     time.Now().Add(-time.Hour).Unix(),
	}).SignedString(testSecret)

	_, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, tok)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expired token was accepted: got %v, want ErrInvalidCredentials", err)
	}
}

// Security: alg=none attack – attacker strips signature and sets algorithm to "none".
// The server must reject this because only HMAC-signed tokens are accepted.
func TestValidateToken_NoneAlgorithmRejected(t *testing.T) {
	noneToken := buildRawJWT("none", map[string]interface{}{
		"user_id": 1,
		"role":    "admin",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}, "")

	_, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, noneToken)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("alg=none token was accepted: got %v, want ErrInvalidCredentials", err)
	}
}

// Security: algorithm confusion attack – attacker crafts a token claiming RS256.
// The server must enforce HS256 and reject all other signing methods.
func TestValidateToken_RSAlgorithmRejected(t *testing.T) {
	rsToken := buildRawJWT("RS256", map[string]interface{}{
		"user_id": 1,
		"role":    "admin",
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}, "fakersasig")

	_, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, rsToken)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("RS256 token was accepted: got %v, want ErrInvalidCredentials", err)
	}
}

// Security: tokens containing an unrecognised role must be rejected.
func TestValidateToken_InvalidRole(t *testing.T) {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(1),
		"role":    "superadmin",
		"exp":     time.Now().Add(time.Hour).Unix(),
	}).SignedString(testSecret)

	_, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, tok)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("token with invalid role was accepted: got %v, want ErrInvalidCredentials", err)
	}
}

func TestValidateToken_MissingUserID(t *testing.T) {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role": domain.RoleUser,
		"exp":  time.Now().Add(time.Hour).Unix(),
	}).SignedString(testSecret)

	_, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, tok)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("token without user_id was accepted: got %v, want ErrInvalidCredentials", err)
	}
}

func TestValidateToken_ValidUserToken(t *testing.T) {
	repo := testhelper.NewMockRepo()
	uc := newUC(repo)
	mustRegister(t, uc, "a@b.com", "pass")
	tok := mustLogin(t, uc, "a@b.com", "pass")

	payload, err := uc.ValidateToken(bg, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Role != domain.RoleUser {
		t.Fatalf("got role %q, want %q", payload.Role, domain.RoleUser)
	}
	if payload.UserID <= 0 {
		t.Fatalf("invalid user_id %d in token payload", payload.UserID)
	}
}

func TestValidateToken_AdminToken(t *testing.T) {
	tok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": float64(7),
		"role":    domain.RoleAdmin,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}).SignedString(testSecret)

	payload, err := newUC(testhelper.NewMockRepo()).ValidateToken(bg, tok)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload.Role != domain.RoleAdmin {
		t.Fatalf("got role %q, want %q", payload.Role, domain.RoleAdmin)
	}
	if payload.UserID != 7 {
		t.Fatalf("got user_id %d, want 7", payload.UserID)
	}
}

// ---- HashPassword -------------------------------------------------------

func TestHashPassword_NotPlaintext(t *testing.T) {
	hashed, err := usecase.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hashed == "mypassword" {
		t.Fatal("hash equals plaintext – password is stored unencrypted")
	}
}

func TestHashPassword_ValidBcrypt(t *testing.T) {
	const plain = "mypassword"
	hashed, err := usecase.HashPassword(plain)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)); err != nil {
		t.Fatalf("produced hash does not verify: %v", err)
	}
}

// bcrypt must use a random salt so two hashes of the same password differ.
// Identical hashes would mean the salt is static, enabling precomputed rainbow-table attacks.
func TestHashPassword_RandomSalt(t *testing.T) {
	h1, _ := usecase.HashPassword("same")
	h2, _ := usecase.HashPassword("same")
	if h1 == h2 {
		t.Fatal("two hashes of the same password are identical – salt is not random")
	}
}

// ---- ChangeRole ---------------------------------------------------------

func TestChangeRole_EmptyEmail(t *testing.T) {
	err := newUC(testhelper.NewMockRepo()).ChangeRole(bg, "", domain.RoleAdmin)
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestChangeRole_InvalidRole(t *testing.T) {
	err := newUC(testhelper.NewMockRepo()).ChangeRole(bg, "a@b.com", "superadmin")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("got %v, want ErrInvalidCredentials", err)
	}
}

func TestChangeRole_ValidRolesAccepted(t *testing.T) {
	for _, role := range []string{domain.RoleUser, domain.RoleAdmin} {
		repo := testhelper.NewMockRepo()
		uc := newUC(repo)
		mustRegister(t, uc, "a@b.com", "pass")
		if err := uc.ChangeRole(bg, "a@b.com", role); err != nil {
			t.Fatalf("ChangeRole(%q) rejected a valid role: %v", role, err)
		}
	}
}

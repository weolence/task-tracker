package middleware

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const UserIDKey contextKey = "user_id"

func AuthMiddleware(secret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractToken(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			token, err := parseAndValidateToken(tokenString, secret)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := extractUserID(token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// saving user_id in context for further use in handlers
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken extracts token from header Authorization
func extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", jwt.ErrSignatureInvalid
	}

	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", jwt.ErrSignatureInvalid
	}

	return authHeader[len(prefix):], nil
}

// parseAndValidateToken parses JWT and checks its validity using the provided secret key
func parseAndValidateToken(tokenString string, secret []byte) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		// checking algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	})

	if err != nil || !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return token, nil
}

// extractUserID extracts and validates user_id from JWT claims
func extractUserID(token *jwt.Token) (any, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	userID, ok := claims["user_id"]
	if !ok {
		return nil, jwt.ErrSignatureInvalid
	}

	return userID, nil
}

// GetUserID extracts user_id from the request context
func GetUserID(ctx context.Context) any {
	userID := ctx.Value(UserIDKey)
	return userID
}

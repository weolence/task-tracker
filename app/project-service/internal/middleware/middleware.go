package middleware

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"project-service/internal/model/dto"

	"google.golang.org/protobuf/encoding/protojson"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func AuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractToken(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			userID, err := validateTokenWithAuthService(tokenString)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (int32, bool) {
	userID, ok := ctx.Value(UserIDKey).(int32)
	return userID, ok
}

func extractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header required")
	}

	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
		return "", errors.New("bad authorization header")
	}

	return authHeader[len(prefix):], nil
}

func validateTokenWithAuthService(token string) (int32, error) {
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "http://localhost:8080"
	}

	reqBody := dto.ValidateTokenRequest{Token: token}
	jsonData, err := protojson.Marshal(&reqBody)
	if err != nil {
		return 0, err
	}

	resp, err := http.Post(authServiceURL+"/validate-token", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var respBody dto.ValidateTokenResponse
	err = protojson.Unmarshal(body, &respBody)
	if err != nil {
		return 0, err
	}

	return respBody.UserId, nil
}

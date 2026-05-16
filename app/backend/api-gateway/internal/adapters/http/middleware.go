package httpadapter

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	authsvcv1 "auth-service/api/proto/authsvcv1"
	userv1 "auth-service/api/proto/userv1"

	"google.golang.org/grpc/metadata"
)

type contextKey string

const (
	ctxUserID   contextKey = "user_id"
	ctxUserRole contextKey = "user_role"
)

// AuthMiddleware validates the Bearer token via the auth-service gRPC endpoint,
// then stores user_id and role in the request context and gRPC metadata for
// forwarding to downstream services.
func AuthMiddleware(auth authsvcv1.AuthServiceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := extractBearer(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			resp, err := auth.ValidateToken(r.Context(), &userv1.ValidateTokenRequest{Token: token})
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ctxUserID, resp.UserId)
			ctx = context.WithValue(ctx, ctxUserRole, resp.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminOnly rejects requests whose role is not "admin".
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, _ := r.Context().Value(ctxUserRole).(string)
		if role != "admin" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// outgoingCtx attaches user identity as gRPC metadata so downstream services
// can read it without re-validating the JWT.
func outgoingCtx(r *http.Request) context.Context {
	userID, _ := r.Context().Value(ctxUserID).(int32)
	role, _ := r.Context().Value(ctxUserRole).(string)
	md := metadata.Pairs(
		"user-id", strconv.Itoa(int(userID)),
		"user-role", role,
	)
	return metadata.NewOutgoingContext(r.Context(), md)
}

func extractBearer(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(auth, prefix) {
		return "", http.ErrNoCookie
	}
	return strings.TrimPrefix(auth, prefix), nil
}

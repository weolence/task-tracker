package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	userv1 "auth-service/api/proto/userv1"
	"project-service/internal/appctx"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserRoleKey contextKey = "user_role"

func AuthMiddleware(authServiceURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenString, err := extractToken(r)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			authData, err := validateTokenWithAuthService(authServiceURL, tokenString)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, authData.UserId)
			ctx = context.WithValue(ctx, UserRoleKey, authData.Role)
			ctx = appctx.WithDBRole(ctx, appRoleToDBRole(authData.Role))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(ctx context.Context) (int32, bool) {
	userID, ok := ctx.Value(UserIDKey).(int32)
	return userID, ok
}

func GetUserRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(UserRoleKey).(string)
	return role, ok
}

func IsAdmin(ctx context.Context) bool {
	role, ok := GetUserRole(ctx)
	return ok && role == "admin"
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r.Context()) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// appRoleToDBRole maps system-level JWT roles to their most restrictive DB role.
// Routes that require manager-level DB access must also apply ProjectRoleMiddleware
// or AsManagerMiddleware to elevate the role when appropriate.
func appRoleToDBRole(appRole string) string {
	if appRole == "admin" {
		return "app_admin"
	}
	return "app_member"
}

// ProjectRoleMiddleware elevates the DB role to app_manager for authenticated
// users who are managers of the project given by the "project_id" query param.
func ProjectRoleMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsAdmin(r.Context()) {
				if userID, ok := GetUserID(r.Context()); ok {
					if s := r.URL.Query().Get("project_id"); s != "" {
						if projectID, err := strconv.Atoi(s); err == nil {
							if isProjectManager(pool, r.Context(), projectID, userID) {
								r = r.WithContext(appctx.WithDBRole(r.Context(), "app_manager"))
							}
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// TaskRoleMiddleware elevates the DB role to app_manager for authenticated users
// who are managers of the project that owns the task identified in the URL path.
// Expected path format: /api/tasks/{taskID}[/...]
func TaskRoleMiddleware(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsAdmin(r.Context()) {
				if userID, ok := GetUserID(r.Context()); ok {
					// /api/tasks/{id}[/...] splits to ["","api","tasks","id",...]
					parts := strings.Split(r.URL.Path, "/")
					if len(parts) >= 4 {
						if taskID, err := strconv.Atoi(parts[3]); err == nil {
							// Resolve the task's project using the pool owner's role
							// (empty DB role in context bypasses SET ROLE in BeforeAcquire)
							var projectID int
							pool.QueryRow(
								appctx.WithDBRole(r.Context(), ""),
								`SELECT project_id FROM tasks WHERE id = $1`,
								taskID,
							).Scan(&projectID)
							if projectID != 0 && isProjectManager(pool, r.Context(), projectID, userID) {
								r = r.WithContext(appctx.WithDBRole(r.Context(), "app_manager"))
							}
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AsManagerMiddleware unconditionally sets the DB role to app_manager for
// non-admin users. Use for endpoints where any authenticated user may perform
// manager-level operations, such as creating a new project.
func AsManagerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r.Context()) {
			r = r.WithContext(appctx.WithDBRole(r.Context(), "app_manager"))
		}
		next.ServeHTTP(w, r)
	})
}

// isProjectManager returns true if userID holds the 'manager' role for projectID.
// The lookup runs with an empty DB role so it uses the pool owner's privileges,
// avoiding the chicken-and-egg problem of determining the role before it is set.
func isProjectManager(pool *pgxpool.Pool, ctx context.Context, projectID int, userID int32) bool {
	var role string
	pool.QueryRow(
		appctx.WithDBRole(ctx, ""),
		`SELECT role FROM project_user_roles WHERE project_id = $1 AND user_id = $2`,
		projectID, userID,
	).Scan(&role)
	return role == "manager"
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

func validateTokenWithAuthService(authServiceURL string, token string) (*userv1.ValidateTokenResponse, error) {
	reqBody := userv1.ValidateTokenRequest{Token: token}
	jsonData, err := protojson.Marshal(&reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(authServiceURL+"/validate-token", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth service returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var respBody userv1.ValidateTokenResponse
	if err := protojson.Unmarshal(body, &respBody); err != nil {
		return nil, err
	}

	if respBody.Role == "" {
		return nil, errors.New("auth service returned empty role")
	}

	return &respBody, nil
}

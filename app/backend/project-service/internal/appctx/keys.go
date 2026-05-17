package appctx

import "context"

type dbRoleKey struct{}

func GetDBRole(ctx context.Context) string {
	role, _ := ctx.Value(dbRoleKey{}).(string)
	return role
}

func WithDBRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, dbRoleKey{}, role)
}

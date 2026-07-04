package auth

import "context"

type contextKey string

const isAdminKey contextKey = "IsAdmin"

func WithAdmin(ctx context.Context, isAdmin bool) context.Context {
	return context.WithValue(ctx, isAdminKey, isAdmin)
}

func IsAdmin(ctx context.Context) bool {
	isAdmin, ok := ctx.Value(isAdminKey).(bool)
	return ok && isAdmin
}

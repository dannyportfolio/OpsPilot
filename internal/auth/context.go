package auth

import (
	"context"

	"opspilot/task-api/internal/platform/tenant"
)

type contextKey string

const userContextKey contextKey = "opspilot.auth.user"

// WithUser adds validated user claims and sets the platform tenant context.
func WithUser(ctx context.Context, claims Claims) context.Context {
	ctx = context.WithValue(ctx, userContextKey, claims)
	if claims.TenantID != "" {
		ctx = tenant.WithTenant(ctx, claims.TenantID)
	}
	return ctx
}

// UserFromContext retrieves authenticated user claims from context.
func UserFromContext(ctx context.Context) (*Claims, bool) {
	val := ctx.Value(userContextKey)
	if val == nil {
		return nil, false
	}
	claims, ok := val.(Claims)
	if !ok {
		return nil, false
	}
	return &claims, true
}

// TenantFromContext retrieves the tenant ID from the context, checking the platform tenant first.
func TenantFromContext(ctx context.Context) (string, bool) {
	if tID, ok := tenant.FromContext(ctx); ok && tID != "" {
		return tID, true
	}
	claims, ok := UserFromContext(ctx)
	if !ok || claims.TenantID == "" {
		return "", false
	}
	return claims.TenantID, true
}

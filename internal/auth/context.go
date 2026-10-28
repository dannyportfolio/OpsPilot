package auth

import "context"

type contextKey string

const userContextKey contextKey = "opspilot.auth.user"

// WithUser adds validated user claims to the request context.
func WithUser(ctx context.Context, claims Claims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
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

// TenantFromContext retrieves the tenant ID from the authenticated user claims.
func TenantFromContext(ctx context.Context) (string, bool) {
	claims, ok := UserFromContext(ctx)
	if !ok || claims.TenantID == "" {
		return "", false
	}
	return claims.TenantID, true
}

package tenant

import "context"

type contextKey struct{}

var tenantKey = contextKey{}

// WithTenant injects an authenticated tenant identifier into the request context.
func WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey, tenantID)
}

// FromContext extracts the tenant identifier from the context if present.
func FromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	val, ok := ctx.Value(tenantKey).(string)
	if !ok || val == "" {
		return "", false
	}
	return val, true
}

package tenant_test

import (
	"context"
	"testing"

	"opspilot/task-api/internal/platform/tenant"
)

func TestTenantContext(t *testing.T) {
	ctx := context.Background()

	// Initial context has no tenant
	if _, ok := tenant.FromContext(ctx); ok {
		t.Error("expected no tenant in empty context")
	}

	// Inject tenant
	tenantCtx := tenant.WithTenant(ctx, "tenant-123")
	got, ok := tenant.FromContext(tenantCtx)
	if !ok {
		t.Fatal("expected tenant to be present")
	}
	if got != "tenant-123" {
		t.Errorf("expected tenant-123, got %q", got)
	}

	// Nil context safety
	if _, ok := tenant.FromContext(nil); ok {
		t.Error("expected no tenant for nil context")
	}
}

package auth_test

import (
	"sync"
	"testing"
	"time"

	"opspilot/task-api/internal/auth"
)

func TestMemoryRevocationStore(t *testing.T) {
	store := auth.NewMemoryRevocationStore()

	jti := "token-jti-123"
	if store.IsRevoked(jti) {
		t.Fatal("expected token to not be revoked initially")
	}

	exp := time.Now().UTC().Add(1 * time.Hour)
	if err := store.Revoke(jti, exp); err != nil {
		t.Fatalf("unexpected error revoking token: %v", err)
	}

	if !store.IsRevoked(jti) {
		t.Fatal("expected token to be marked as revoked")
	}

	// Already expired revocation should be treated as not active
	expiredJTI := "expired-jti"
	_ = store.Revoke(expiredJTI, time.Now().UTC().Add(-1*time.Minute))
	if store.IsRevoked(expiredJTI) {
		t.Fatal("expected expired token to not report as active revocation")
	}
}

func TestMemoryRevocationStore_Concurrency(t *testing.T) {
	store := auth.NewMemoryRevocationStore()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			jti := "concurrent-jti"
			_ = store.Revoke(jti, time.Now().UTC().Add(10*time.Minute))
			_ = store.IsRevoked(jti)
		}(i)
	}

	wg.Wait()
}

package auth

import (
	"sync"
	"time"
)

// RevocationStore tracks revoked token identifiers.
type RevocationStore interface {
	Revoke(jti string, expiresAt time.Time) error
	IsRevoked(jti string) bool
}

// MemoryRevocationStore is a thread-safe in-memory store for revoked token identifiers.
type MemoryRevocationStore struct {
	mu      sync.RWMutex
	revoked map[string]time.Time
}

// NewMemoryRevocationStore initializes a new in-memory revocation store.
func NewMemoryRevocationStore() *MemoryRevocationStore {
	return &MemoryRevocationStore{
		revoked: make(map[string]time.Time),
	}
}

// Revoke records a token's JTI as revoked until its natural expiration.
func (s *MemoryRevocationStore) Revoke(jti string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clean up already expired tokens to prevent unbounded memory growth
	now := time.Now().UTC()
	for k, exp := range s.revoked {
		if exp.Before(now) {
			delete(s.revoked, k)
		}
	}

	s.revoked[jti] = expiresAt
	return nil
}

// IsRevoked checks if a token's JTI is in the revocation store and not yet expired.
func (s *MemoryRevocationStore) IsRevoked(jti string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	exp, found := s.revoked[jti]
	if !found {
		return false
	}

	// If the expiration passed, it is no longer valid anyway
	if exp.Before(time.Now().UTC()) {
		return false
	}

	return true
}

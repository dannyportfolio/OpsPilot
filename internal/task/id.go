package task

import (
	"crypto/rand"
	"fmt"
	"io"
)

// GenerateID produces a cryptographically secure RFC 4122 v4 UUID string.
// It avoids external dependencies while strictly conforming to UUIDv4 specifications.
func GenerateID() (string, error) {
	var uuid [16]byte
	if _, err := io.ReadFull(rand.Reader, uuid[:]); err != nil {
		return "", fmt.Errorf("failed to generate random id: %w", err)
	}

	// Set version 4 (0100)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant 10xx
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16],
	), nil
}

// IsValidID validates whether the string strictly conforms to RFC 4122 UUID format (36 chars: 8-4-4-4-12 hex).
func IsValidID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, c := range id {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}

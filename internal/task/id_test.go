package task_test

import (
	"regexp"
	"testing"

	"opspilot/task-api/internal/task"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestGenerateID(t *testing.T) {
	seen := make(map[string]bool)
	const count = 1000

	for i := 0; i < count; i++ {
		id, err := task.GenerateID()
		if err != nil {
			t.Fatalf("unexpected error generating ID: %v", err)
		}

		if !uuidRegex.MatchString(id) {
			t.Errorf("generated ID %q does not match RFC 4122 v4 UUID format", id)
		}

		if seen[id] {
			t.Fatalf("collision detected on iteration %d: %q", i, id)
		}
		seen[id] = true
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"c1f10842-83fe-43dc-9e87-a2f073285709", true},
		{"00000000-0000-0000-0000-000000000000", true},
		{"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", true},
		{"", false},
		{"123", false},
		{"not-a-valid-uuid", false},
		{"c1f10842-83fe-43dc-9e87-a2f07328570g", false},  // 'g' is non-hex
		{"c1f1084283fe43dc9e87a2f073285709", false},      // missing dashes
		{"c1f10842-83fe-43dc-9e87-a2f0732857090", false}, // too long
		{"c1f10842-83fe-43dc-9e87-a2f07328570", false},   // too short
	}

	for _, tt := range tests {
		got := task.IsValidID(tt.id)
		if got != tt.want {
			t.Errorf("IsValidID(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

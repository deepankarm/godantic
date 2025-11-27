package partialjson

import (
	"testing"
)

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name string
		path []string
		want string
	}{
		{"empty", []string{}, ""},
		{"single", []string{"name"}, "name"},
		{"two parts", []string{"user", "name"}, "user.name"},
		{"three parts", []string{"user", "address", "city"}, "user.address.city"},
		{"with array index", []string{"items", "[0]", "name"}, "items[0].name"},
		{"array at root", []string{"[0]"}, "[0]"},
		{"multiple arrays", []string{"users", "[0]", "orders", "[1]", "total"}, "users[0].orders[1].total"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinPath(tt.path)
			if got != tt.want {
				t.Errorf("JoinPath(%v) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildIncompleteSet(t *testing.T) {
	paths := [][]string{
		{"name"},
		{"user", "email"},
		{"items", "[0]", "title"},
	}

	set := BuildIncompleteSet(paths)

	// Should contain all paths
	expected := []string{"name", "user.email", "items[0].title"}
	for _, p := range expected {
		if !set[p] {
			t.Errorf("expected set to contain %q", p)
		}
	}

	// Should not contain other paths
	notExpected := []string{"user", "email", "items", "items[0]"}
	for _, p := range notExpected {
		if set[p] {
			t.Errorf("expected set NOT to contain %q", p)
		}
	}
}

func TestIsPathOrParentIncomplete(t *testing.T) {
	incompleteSet := map[string]bool{
		"name":           true,
		"user.email":     true,
		"items[0].title": true,
	}

	tests := []struct {
		path string
		want bool
	}{
		// Direct matches
		{"name", true},
		{"user.email", true},
		{"items[0].title", true},

		// Parent incomplete
		{"name.extra", true},          // "name" is incomplete
		{"user.email.domain", true},   // "user.email" is incomplete
		{"items[0].title.text", true}, // "items[0].title" is incomplete

		// Not incomplete
		{"age", false},
		{"user", false},             // Parent of "user.email" but not itself incomplete
		{"user.name", false},        // Different child of "user"
		{"items", false},            // Parent of "items[0].title"
		{"items[0]", false},         // Parent of "items[0].title"
		{"items[1].title", false},   // Different array index
		{"other.field.path", false}, // Completely different
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsPathOrParentIncomplete(tt.path, incompleteSet)
			if got != tt.want {
				t.Errorf("IsPathOrParentIncomplete(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

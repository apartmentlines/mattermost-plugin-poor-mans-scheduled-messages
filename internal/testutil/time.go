package testutil

import (
	"testing"
	"time"
)

func MustLoadLocation(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("Failed to load location %q: %v", name, err)
	}
	return loc
}

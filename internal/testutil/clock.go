// Package testutil provides helpers for tests.
package testutil

import "time"

// FakeClock is a controllable clock for tests.
type FakeClock struct{ NowTime time.Time }

// Now returns the configured time.
func (f FakeClock) Now() time.Time { return f.NowTime }

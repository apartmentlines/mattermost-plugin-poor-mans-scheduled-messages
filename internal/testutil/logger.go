package testutil

// FakeLogger is a no-op logger for tests.
type FakeLogger struct{}

// Error is a no-op error log.
func (FakeLogger) Error(string, ...any) {}

// Warn is a no-op warning log.
func (FakeLogger) Warn(string, ...any) {}

// Info is a no-op info log.
func (FakeLogger) Info(string, ...any) {}

// Debug is a no-op debug log.
func (FakeLogger) Debug(string, ...any) {}

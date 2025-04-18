package testutil

type FakeLogger struct{}

func (FakeLogger) Error(string, ...any) {}
func (FakeLogger) Warn(string, ...any)  {}
func (FakeLogger) Info(string, ...any)  {}
func (FakeLogger) Debug(string, ...any) {}

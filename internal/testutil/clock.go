package testutil

import "time"

type FakeClock struct{ NowTime time.Time }

func (f FakeClock) Now() time.Time { return f.NowTime }

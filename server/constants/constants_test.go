package constants

import "testing"

func TestConstantValues(t *testing.T) {
	if SchedPrefix != "schedmsg:" {
		t.Fatalf("SchedPrefix unexpected: %s", SchedPrefix)
	}
	if UserIndexPrefix != "user_sched_index:" {
		t.Fatalf("UserIndexPrefix unexpected: %s", UserIndexPrefix)
	}
}

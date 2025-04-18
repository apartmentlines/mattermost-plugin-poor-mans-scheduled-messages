package testutil

import "fmt"

const (
	// SchedPrefix is the prefix used for scheduled message keys in the KV store.
	SchedPrefix = "schedmsg:"
	// UserIndexPrefix is the prefix used for user message index keys in the KV store.
	UserIndexPrefix = "user_sched_index:"
	// MaxUserMessages is a common limit used in tests involving user message counts.
	MaxUserMessages = 1000
)

func SchedKey(id string) string {
	return fmt.Sprintf("%s%s", SchedPrefix, id)
}

func IndexKey(userID string) string {
	return fmt.Sprintf("%s%s", UserIndexPrefix, userID)
}

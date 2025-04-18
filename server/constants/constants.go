package constants

const (
	// SchedPrefix is the prefix used for scheduled message keys in the KV store.
	SchedPrefix = "schedmsg:"
	// UserIndexPrefix is the prefix used for user message index keys in the KV store.
	UserIndexPrefix = "user_sched_index:"
	// MaxUserMessages is a common limit used in tests involving user message counts.
	MaxUserMessages = 1000
)

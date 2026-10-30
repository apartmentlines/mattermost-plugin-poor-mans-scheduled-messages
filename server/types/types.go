//revive:disable:var-naming // Package name is conventional for shared types.
package types

import "time"

// ScheduledMessage represents a message scheduled for future delivery.
type ScheduledMessage struct {
	ID             string        `json:"id"`
	UserID         string        `json:"user_id"`
	ChannelID      string        `json:"channel_id"`
	RootID         string        `json:"root_id,omitempty"`
	PostAt         time.Time     `json:"post_at"`
	MessageContent string        `json:"message_content"`
	Timezone       string        `json:"timezone"`
	State          DeliveryState `json:"state,omitempty"`
	Attempts       int           `json:"attempts,omitempty"`
	ClaimToken     string        `json:"claim_token,omitempty"`
	LeaseUntil     time.Time     `json:"lease_until,omitzero"`
	NextAttemptAt  time.Time     `json:"next_attempt_at,omitzero"`
	PostID         string        `json:"post_id,omitempty"`
	LastReason     string        `json:"last_reason,omitempty"`
}

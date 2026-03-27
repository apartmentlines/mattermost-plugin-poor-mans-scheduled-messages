//revive:disable:var-naming // Package name is conventional for shared types.
package types

import "time"

// ScheduledMessage represents a message scheduled for future delivery.
type ScheduledMessage struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ChannelID      string    `json:"channel_id"`
	RootPostID     string    `json:"root_post_id,omitempty"`
	PostAt         time.Time `json:"post_at"`
	MessageContent string    `json:"message_content"`
	Timezone       string    `json:"timezone"`
}

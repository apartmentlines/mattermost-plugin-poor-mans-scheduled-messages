package types

import "time"

type Logger interface {
	Error(msg string, keyvals ...any)
	Warn(msg string, keyvals ...any)
	Info(msg string, keyvals ...any)
	Debug(msg string, keyvals ...any)
}

type ScheduledMessage struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	ChannelID      string    `json:"channel_id"`
	PostAt         time.Time `json:"post_at"`
	MessageContent string    `json:"message_content"`
	Timezone       string    `json:"timezone"`
}

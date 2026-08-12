//revive:disable:var-naming // Package name is conventional for shared types.
package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestScheduledMessageJSONRoundTrip(t *testing.T) {
	original := ScheduledMessage{
		ID:             "id1",
		UserID:         "user1",
		ChannelID:      "channel1",
		RootID:         "root1",
		PostAt:         time.Unix(1700000000, 0).UTC(),
		MessageContent: "hello",
		Timezone:       "UTC",
	}

	data, err := json.Marshal(&original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ScheduledMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if original.ID != decoded.ID ||
		original.UserID != decoded.UserID ||
		original.ChannelID != decoded.ChannelID ||
		original.RootID != decoded.RootID ||
		!original.PostAt.Equal(decoded.PostAt) ||
		original.MessageContent != decoded.MessageContent ||
		original.Timezone != decoded.Timezone {
		t.Fatalf("round‑trip mismatch: expected %+v got %+v", original, decoded)
	}
}

func TestScheduledMessageJSONMissingRootIDDecodesAsEmpty(t *testing.T) {
	data := []byte(`{
		"id":"id1",
		"user_id":"user1",
		"channel_id":"channel1",
		"post_at":"2023-11-14T22:13:20Z",
		"message_content":"hello",
		"timezone":"UTC"
	}`)

	var decoded ScheduledMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal legacy record: %v", err)
	}

	if decoded.RootID != "" {
		t.Fatalf("expected legacy record to have no root ID, got %q", decoded.RootID)
	}
}

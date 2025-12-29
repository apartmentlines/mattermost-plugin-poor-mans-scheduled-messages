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
		!original.PostAt.Equal(decoded.PostAt) ||
		original.MessageContent != decoded.MessageContent ||
		original.Timezone != decoded.Timezone {
		t.Fatalf("round‑trip mismatch: expected %+v got %+v", original, decoded)
	}
}

// --- File: pkg/store/kv.go ---
package store

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
)

type Store interface {
	SaveScheduledMessage(ctx context.Context, msg *types.ScheduledMessage) error
	DeleteScheduledMessage(ctx context.Context, msgID string) error
	GetScheduledMessage(ctx context.Context, msgID string) (*types.ScheduledMessage, error)
	ListAllScheduledIDs(ctx context.Context) (map[string]int64, error)
	ListUserMessageIDs(ctx context.Context, userID string) ([]string, error)
	AddUserMessageID(ctx context.Context, userID, msgID string) error
	RemoveUserMessageID(ctx context.Context, userID, msgID string) error
	GenerateMessageID() string
}

type kvStore struct {
	kv *pluginapi.KVService
}

func NewKVStore(kv *pluginapi.KVService) Store {
	return &kvStore{kv: kv}
}

func (s *kvStore) SaveScheduledMessage(ctx context.Context, msg *types.ScheduledMessage) error {
	key := fmt.Sprintf("schedmsg:%s", msg.ID)
	return s.kv.Set(ctx, key, msg)
}

func (s *kvStore) DeleteScheduledMessage(ctx context.Context, msgID string) error {
	key := fmt.Sprintf("schedmsg:%s", msgID)
	return s.kv.Delete(ctx, key)
}

func (s *kvStore) GetScheduledMessage(ctx context.Context, msgID string) (*types.ScheduledMessage, error) {
	var msg types.ScheduledMessage
	key := fmt.Sprintf("schedmsg:%s", msgID)
	err := s.kv.Get(ctx, key, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *kvStore) ListAllScheduledIDs(ctx context.Context) (map[string]int64, error) {
	results := make(map[string]int64)
	keys, err := s.kv.ListKeys(ctx, pluginapi.WithPrefix("schedmsg:"))
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		var msg types.ScheduledMessage
		err := s.kv.Get(ctx, key, &msg)
		if err == nil {
			results[msg.ID] = msg.PostAt.Unix()
		}
	}
	return results, nil
}

func (s *kvStore) ListUserMessageIDs(ctx context.Context, userID string) ([]string, error) {
	var ids []string
	key := fmt.Sprintf("user_sched_index:%s", userID)
	err := s.kv.Get(ctx, key, &ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *kvStore) AddUserMessageID(ctx context.Context, userID, msgID string) error {
	key := fmt.Sprintf("user_sched_index:%s", userID)
	var ids []string
	s.kv.Get(ctx, key, &ids)
	for _, id := range ids {
		if id == msgID {
			return nil
		}
	}
	ids = append(ids, msgID)
	return s.kv.Set(ctx, key, ids)
}

func (s *kvStore) RemoveUserMessageID(ctx context.Context, userID, msgID string) error {
	key := fmt.Sprintf("user_sched_index:%s", userID)
	var ids []string
	s.kv.Get(ctx, key, &ids)
	out := ids[:0]
	for _, id := range ids {
		if id != msgID {
			out = append(out, id)
		}
	}
	return s.kv.Set(ctx, key, out)
}

func (s *kvStore) GenerateMessageID() string {
	return uuid.NewString()
}

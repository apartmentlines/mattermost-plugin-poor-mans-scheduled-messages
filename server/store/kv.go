package store

import (
	"fmt"
	"slices"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Store interface {
	SaveScheduledMessage(userID string, msg *types.ScheduledMessage) error
	DeleteScheduledMessage(userID string, msgID string) error
	CleanupMessageFromUserIndex(userID string, msgID string) error
	GetScheduledMessage(msgID string) (*types.ScheduledMessage, error)
	ListAllScheduledIDs() (map[string]int64, error)
	ListUserMessageIDs(userID string) ([]string, error)
	GenerateMessageID() string
}

type kvStore struct {
	client *pluginapi.Client
}

func NewKVStore(client *pluginapi.Client) Store {
	return &kvStore{client: client}
}

func (s *kvStore) SaveScheduledMessage(userID string, msg *types.ScheduledMessage) error {
	_, addIndexErr := s.addUserMessageToIndex(userID, msg.ID)
	if addIndexErr != nil {
		return addIndexErr
	}
	_, saveMessageErr := s.saveNewScheduledMessage(msg)
	if saveMessageErr != nil {
		return saveMessageErr
	}
	return nil
}

func (s *kvStore) DeleteScheduledMessage(userID string, msgID string) error {
	scheduleErr := s.deleteScheduledMessageByID(msgID)
	if scheduleErr != nil {
		return scheduleErr
	}
	_, removeIndexErr := s.removeUserMessageFromIndex(userID, msgID)
	if removeIndexErr != nil {
		return removeIndexErr
	}
	return nil
}

func (s *kvStore) CleanupMessageFromUserIndex(userID string, msgID string) error {
	_, removeIndexErr := s.removeUserMessageFromIndex(userID, msgID)
	if removeIndexErr != nil {
		return removeIndexErr
	}
	return nil
}

func (s *kvStore) GetScheduledMessage(msgID string) (*types.ScheduledMessage, error) {
	var msg types.ScheduledMessage
	key := fmt.Sprintf("schedmsg:%s", msgID)
	err := s.client.KV.Get(key, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *kvStore) ListAllScheduledIDs() (map[string]int64, error) {
	results := make(map[string]int64)
	// TODO: Make the count a constant, don't let a user schedule more than 1000 messages.
	keys, err := s.client.KV.ListKeys(0, 1000, pluginapi.WithPrefix("schedmsg:"))
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		var msg types.ScheduledMessage
		err := s.client.KV.Get(key, &msg)
		if err == nil {
			results[msg.ID] = msg.PostAt.Unix()
		}
	}
	return results, nil
}

func (s *kvStore) ListUserMessageIDs(userID string) ([]string, error) {
	var ids []string
	key := fmt.Sprintf("user_sched_index:%s", userID)
	err := s.client.KV.Get(key, &ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *kvStore) GenerateMessageID() string {
	return uuid.NewString()
}

func (s *kvStore) removeUserMessageFromIndex(userID, msgID string) (bool, error) {
	key := fmt.Sprintf("user_sched_index:%s", userID)
	var ids []string
	getErr := s.client.KV.Get(key, &ids)
	if getErr != nil {
		return false, getErr
	}
	i := slices.Index(ids, msgID)
	if i != -1 {
		ids = slices.Delete(ids, i, i+1)
	}
	return s.client.KV.Set(key, ids)
}

func (s *kvStore) addUserMessageToIndex(userID, msgID string) (bool, error) {
	key := fmt.Sprintf("user_sched_index:%s", userID)
	var ids []string
	getErr := s.client.KV.Get(key, &ids)
	if getErr != nil {
		return false, getErr
	}
	if slices.Contains(ids, msgID) {
		return true, nil
	}
	ids = append(ids, msgID)
	return s.client.KV.Set(key, ids)
}

func (s *kvStore) saveNewScheduledMessage(msg *types.ScheduledMessage) (bool, error) {
	key := fmt.Sprintf("schedmsg:%s", msg.ID)
	return s.client.KV.Set(key, msg)
}

func (s *kvStore) deleteScheduledMessageByID(msgID string) error {
	key := fmt.Sprintf("schedmsg:%s", msgID)
	return s.client.KV.Delete(key)
}

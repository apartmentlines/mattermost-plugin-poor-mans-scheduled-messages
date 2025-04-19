package store

import (
	"fmt"
	"slices"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/google/uuid"
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
	logger              ports.Logger
	kv                  ports.KVService
	listMatchingService ports.ListMatchingService
	maxUserMessages     int
}

func NewKVStore(logger ports.Logger, kv ports.KVService, listMatchingService ports.ListMatchingService, maxUserMessages int) Store {
	return &kvStore{logger: logger, kv: kv, listMatchingService: listMatchingService, maxUserMessages: maxUserMessages}
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
	key := schedKey(msgID)
	err := s.kv.Get(key, &msg)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *kvStore) ListAllScheduledIDs() (map[string]int64, error) {
	results := make(map[string]int64)
	keys, err := s.kv.ListKeys(0, s.maxUserMessages, s.listMatchingService.WithPrefix(constants.SchedPrefix))
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		var msg types.ScheduledMessage
		err := s.kv.Get(key, &msg)
		if err == nil {
			results[msg.ID] = msg.PostAt.Unix()
		}
	}
	return results, nil
}

func (s *kvStore) ListUserMessageIDs(userID string) ([]string, error) {
	var ids []string
	key := indexKey(userID)
	err := s.kv.Get(key, &ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *kvStore) GenerateMessageID() string {
	return uuid.NewString()
}

func (s *kvStore) removeUserMessageFromIndex(userID, msgID string) (bool, error) {
	return s.modifyUserIndex(userID, func(ids []string) ([]string, bool) {
		idx := slices.Index(ids, msgID)
		if idx == -1 {
			return ids, false
		}
		return slices.Delete(ids, idx, idx+1), true
	})
}

func (s *kvStore) addUserMessageToIndex(userID, msgID string) (bool, error) {
	return s.modifyUserIndex(userID, func(ids []string) ([]string, bool) {
		if slices.Contains(ids, msgID) {
			return ids, false
		}
		return append(ids, msgID), true
	})
}

func (s *kvStore) saveNewScheduledMessage(msg *types.ScheduledMessage) (bool, error) {
	key := schedKey(msg.ID)
	return s.kv.Set(key, msg)
}

func (s *kvStore) deleteScheduledMessageByID(msgID string) error {
	key := schedKey(msgID)
	return s.kv.Delete(key)
}

func (s *kvStore) modifyUserIndex(
	userID string,
	fn func([]string) ([]string, bool),
) (bool, error) {
	key := indexKey(userID)
	var ids []string
	if err := s.kv.Get(key, &ids); err != nil {
		return false, err
	}
	newIDs, modified := fn(ids)
	if !modified {
		return false, nil
	}
	return s.kv.Set(key, newIDs)
}

func schedKey(id string) string {
	return fmt.Sprintf("%s%s", constants.SchedPrefix, id)
}

func indexKey(userID string) string {
	return fmt.Sprintf("%s%s", constants.UserIndexPrefix, userID)
}

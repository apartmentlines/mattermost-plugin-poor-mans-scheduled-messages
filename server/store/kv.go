package store

import (
	"errors"
	"fmt"
	"slices"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/google/uuid"
)

type kvStore struct {
	logger              ports.Logger
	kv                  ports.KVService
	listMatchingService ports.ListMatchingService
	maxUserMessages     int
}

func NewKVStore(logger ports.Logger, kv ports.KVService, listMatchingService ports.ListMatchingService, maxUserMessages int) ports.Store {
	logger.Debug("Creating new KVStore instance")
	return &kvStore{logger: logger, kv: kv, listMatchingService: listMatchingService, maxUserMessages: maxUserMessages}
}

func (s *kvStore) SaveScheduledMessage(userID string, msg *types.ScheduledMessage) error {
	s.logger.Debug("Attempting to save scheduled message", "user_id", userID, "message_id", msg.ID)

	s.logger.Debug("Adding message ID to user index", "user_id", userID, "message_id", msg.ID)
	_, addIndexErr := s.addUserMessageToIndex(userID, msg.ID)
	if addIndexErr != nil {
		s.logger.Error("Failed to add message ID to user index", "user_id", userID, "message_id", msg.ID, "error", addIndexErr)
		return fmt.Errorf("failed to update user index: %w", addIndexErr)
	}
	s.logger.Debug("Successfully added message ID to user index", "user_id", userID, "message_id", msg.ID)

	s.logger.Debug("Saving scheduled message data", "message_id", msg.ID)
	_, saveMessageErr := s.saveNewScheduledMessage(msg)
	if saveMessageErr != nil {
		s.logger.Error("Failed to save scheduled message data", "message_id", msg.ID, "error", saveMessageErr)
		return fmt.Errorf("failed to save message data: %w", saveMessageErr)
	}
	s.logger.Info("Successfully saved scheduled message and updated index", "user_id", userID, "message_id", msg.ID)
	return nil
}

func (s *kvStore) DeleteScheduledMessage(userID string, msgID string) error {
	s.logger.Debug("Attempting to delete scheduled message", "user_id", userID, "message_id", msgID)

	s.logger.Debug("Deleting scheduled message data", "message_id", msgID)
	scheduleErr := s.deleteScheduledMessageByID(msgID)
	if scheduleErr != nil {
		s.logger.Error("Failed to delete scheduled message data", "message_id", msgID, "error", scheduleErr)
		return fmt.Errorf("failed to delete message data: %w", scheduleErr)
	}
	s.logger.Debug("Successfully deleted scheduled message data", "message_id", msgID)

	s.logger.Debug("Removing message ID from user index", "user_id", userID, "message_id", msgID)
	_, removeIndexErr := s.removeUserMessageFromIndex(userID, msgID)
	if removeIndexErr != nil {
		s.logger.Error("Failed to remove message ID from user index", "user_id", userID, "message_id", msgID, "error", removeIndexErr)
		return fmt.Errorf("failed to remove from user index: %w", removeIndexErr)
	}
	s.logger.Debug("Successfully removed message ID from user index", "user_id", userID, "message_id", msgID)
	s.logger.Info("Successfully deleted scheduled message and removed from index", "user_id", userID, "message_id", msgID)
	return nil
}

func (s *kvStore) CleanupMessageFromUserIndex(userID string, msgID string) error {
	s.logger.Debug("Attempting to cleanup message ID from user index", "user_id", userID, "message_id", msgID)
	_, removeIndexErr := s.removeUserMessageFromIndex(userID, msgID)
	if removeIndexErr != nil {
		s.logger.Error("Failed to remove message ID from user index during cleanup", "user_id", userID, "message_id", msgID, "error", removeIndexErr)
		return fmt.Errorf("failed cleanup remove from user index: %w", removeIndexErr)
	}
	s.logger.Debug("Successfully cleaned up message ID from user index (or it was already gone)", "user_id", userID, "message_id", msgID)
	return nil
}

func (s *kvStore) GetScheduledMessage(msgID string) (*types.ScheduledMessage, error) {
	s.logger.Debug("Attempting to get scheduled message", "message_id", msgID)
	var msg types.ScheduledMessage
	key := schedKey(msgID)
	s.logger.Debug("Calling KV Get", "key", key)
	err := s.kv.Get(key, &msg)
	if err != nil {
		s.logger.Error("Failed to get scheduled message from KV store", "key", key, "error", err)
		return nil, fmt.Errorf("kv.Get failed for key %s: %w", key, err)
	}
	if msg.ID == "" {
		s.logger.Debug("message not found (possibly already sent)", "message_id", msgID, "key", key)
		return nil, errors.New("message not found (possibly already sent)")
	}
	s.logger.Debug("Successfully retrieved scheduled message", "message_id", msgID, "key", key)
	return &msg, nil
}

func (s *kvStore) ListScheduledMessages() ([]*types.ScheduledMessage, error) {
	s.logger.Debug("Attempting to list all scheduled messages")
	var messages []*types.ScheduledMessage
	prefix := constants.SchedPrefix
	s.logger.Debug("Calling KV ListKeys", "prefix", prefix, "page", constants.DefaultPage, "perPage", constants.MaxFetchScheduledMessages)
	keys, err := s.kv.ListKeys(constants.DefaultPage, constants.MaxFetchScheduledMessages, s.listMatchingService.WithPrefix(prefix))
	if err != nil {
		s.logger.Error("Failed to list keys from KV store", "prefix", prefix, "error", err)
		return nil, fmt.Errorf("kv.ListKeys failed for prefix %s: %w", prefix, err)
	}
	s.logger.Debug("Successfully listed keys", "prefix", prefix, "count", len(keys))

	getFailedCount := 0
	for _, key := range keys {
		var msg types.ScheduledMessage
		getErr := s.kv.Get(key, &msg)
		if getErr != nil {
			s.logger.Warn("Failed to get individual scheduled message during list operation", "key", key, "error", getErr)
			getFailedCount++
			continue
		}
		if msg.ID == "" { // Should not happen if key exists, but good to check
			s.logger.Warn("Retrieved message with empty ID during list operation", "key", key)
			getFailedCount++
			continue
		}
		messages = append(messages, &msg)
	}
	s.logger.Debug("Finished processing keys for ListScheduledMessages", "total_keys", len(keys), "successful_gets", len(messages), "failed_gets", getFailedCount)
	return messages, nil
}

func (s *kvStore) ListUserMessageIDs(userID string) ([]string, error) {
	s.logger.Debug("Attempting to list user message IDs", "user_id", userID)
	var ids []string
	key := indexKey(userID)
	s.logger.Debug("Calling KV Get for user index", "key", key)
	err := s.kv.Get(key, &ids)
	if err != nil {
		s.logger.Error("Failed to get user message index from KV store", "key", key, "error", err)
		return nil, fmt.Errorf("kv.Get failed for user index key %s: %w", key, err)
	}
	s.logger.Debug("Successfully retrieved user message index", "user_id", userID, "key", key, "count", len(ids))
	return ids, nil
}

func (s *kvStore) GenerateMessageID() string {
	id := uuid.NewString()
	s.logger.Debug("Generated new message ID", "message_id", id)
	return id
}

func (s *kvStore) removeUserMessageFromIndex(userID, msgID string) (bool, error) {
	s.logger.Debug("Calling modifyUserIndex to remove message ID", "user_id", userID, "message_id", msgID)
	return s.modifyUserIndex(userID, func(ids []string) ([]string, bool) {
		idx := slices.Index(ids, msgID)
		if idx == -1 {
			s.logger.Warn("Message ID not found in user index for removal", "user_id", userID, "message_id", msgID)
			return ids, false
		}
		s.logger.Debug("Found message ID in index, preparing removal", "user_id", userID, "message_id", msgID, "index", idx)
		return slices.Delete(ids, idx, idx+1), true
	})
}

func (s *kvStore) addUserMessageToIndex(userID, msgID string) (bool, error) {
	s.logger.Debug("Calling modifyUserIndex to add message ID", "user_id", userID, "message_id", msgID)
	return s.modifyUserIndex(userID, func(ids []string) ([]string, bool) {
		if slices.Contains(ids, msgID) {
			s.logger.Warn("Message ID already exists in user index", "user_id", userID, "message_id", msgID)
			return ids, false
		}
		s.logger.Debug("Message ID not in index, preparing addition", "user_id", userID, "message_id", msgID)
		return append(ids, msgID), true
	})
}

func (s *kvStore) saveNewScheduledMessage(msg *types.ScheduledMessage) (bool, error) {
	key := schedKey(msg.ID)
	s.logger.Debug("Calling KV Set to save scheduled message", "key", key, "message_id", msg.ID)
	set, err := s.kv.Set(key, msg)
	if err != nil {
		s.logger.Error("Failed to set scheduled message in KV store", "key", key, "message_id", msg.ID, "error", err)
		return false, fmt.Errorf("kv.Set failed for key %s: %w", key, err)
	}
	s.logger.Debug("Successfully set scheduled message in KV store", "key", key, "message_id", msg.ID, "set_result", set)
	return set, nil
}

func (s *kvStore) deleteScheduledMessageByID(msgID string) error {
	key := schedKey(msgID)
	s.logger.Debug("Calling KV Delete for scheduled message", "key", key, "message_id", msgID)
	err := s.kv.Delete(key)
	if err != nil {
		s.logger.Error("Failed to delete scheduled message from KV store", "key", key, "message_id", msgID, "error", err)
		return fmt.Errorf("kv.Delete failed for key %s: %w", key, err)
	}
	s.logger.Debug("Successfully deleted scheduled message from KV store", "key", key, "message_id", msgID)
	return nil
}

func (s *kvStore) modifyUserIndex(
	userID string,
	fn func([]string) ([]string, bool),
) (bool, error) {
	key := indexKey(userID)
	s.logger.Debug("Modifying user index", "user_id", userID, "key", key)

	var ids []string
	s.logger.Debug("Getting current user index from KV", "key", key)
	if err := s.kv.Get(key, &ids); err != nil {
		s.logger.Warn("Failed to get user index", "key", key, "error", err)
		return false, fmt.Errorf("kv.Get failed for index key %s: %w", key, err)
	}
	s.logger.Debug("Successfully retrieved current user index", "key", key, "count", len(ids))

	s.logger.Debug("Applying modification function to index data", "key", key)
	newIDs, modified := fn(ids)
	if !modified {
		s.logger.Debug("Index modification function indicated no changes needed", "key", key)
		return false, nil
	}

	s.logger.Debug("Index was modified, calling KV Set to save updated index", "key", key, "new_count", len(newIDs))
	set, err := s.kv.Set(key, newIDs)
	if err != nil {
		s.logger.Error("Failed to set updated user index in KV store", "key", key, "error", err)
		return false, fmt.Errorf("kv.Set failed for user index key %s: %w", key, err)
	}
	s.logger.Debug("Successfully updated user index in KV store", "key", key, "set_result", set)
	return set, nil
}

func schedKey(id string) string {
	return fmt.Sprintf("%s%s", constants.SchedPrefix, id)
}

func indexKey(userID string) string {
	return fmt.Sprintf("%s%s", constants.UserIndexPrefix, userID)
}

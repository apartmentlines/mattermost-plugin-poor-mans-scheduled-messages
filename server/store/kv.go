// Package store provides persistence for scheduled messages.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

const maxCASAttempts = 64

type indexChange uint8

const (
	indexReserve indexChange = iota
	indexRepair
	indexRemove
)

type kvStore struct {
	logger          ports.Logger
	kv              ports.KVService
	maxUserMessages int
}

// NewKVStore constructs a durable job store. CAS compares original bytes,
// including for legacy JSON records written before state tracking existed.
func NewKVStore(logger ports.Logger, kv ports.KVService, _ ports.ListMatchingService, maxUserMessages int) ports.Store {
	return &kvStore{logger: logger, kv: kv, maxUserMessages: maxUserMessages}
}

func (s *kvStore) SaveScheduledMessage(userID string, msg *types.ScheduledMessage) error {
	if msg == nil || msg.ID == "" || msg.UserID != userID {
		return types.ErrNotOwner
	}
	// Persist a non-publishable record before reserving index capacity. Cleanup
	// can now distinguish a live creation from an abandoned index entry.
	copyMsg := *msg
	copyMsg.State = types.StateCreating
	copyMsg.Attempts = 0
	clearClaim(&copyMsg)
	copyMsg.PostID = ""
	copyMsg.LastReason = ""
	written, err := s.kv.Set(schedKey(msg.ID), &copyMsg, pluginapi.SetAtomic(nil))
	if err != nil {
		return fmt.Errorf("failed to save job: %w", err)
	}
	if !written {
		return types.ErrConflict
	}
	_, err = s.activateCreation(userID, msg.ID)
	return err
}

func (s *kvStore) activateCreation(userID, msgID string) (*types.ScheduledMessage, error) {
	if err := s.modifyUserIndex(userID, msgID, indexReserve); err != nil {
		if errors.Is(err, types.ErrMessageFinal) {
			// Recovery can deliver the job before its creator resumes.
			msg, loadErr := s.GetScheduledMessage(msgID)
			if loadErr != nil {
				return nil, loadErr
			}
			if msg.UserID != userID {
				return nil, types.ErrNotOwner
			}
			if msg.State == types.StateSent {
				return msg, nil
			}
			return nil, err
		}
		if !errors.Is(err, types.ErrMessageLimit) {
			return nil, fmt.Errorf("job %s is retained for creation recovery: %w", msgID, err)
		}
		// Reject only an unaccepted creation. A concurrent recovery may already
		// have activated the job; never cancel a successfully accepted job.
		msg, updateErr := s.update(msgID, func(msg *types.ScheduledMessage) error {
			if msg.UserID != userID {
				return types.ErrNotOwner
			}
			if msg.State == types.StateCreating {
				msg.State = types.StateCancelled
				msg.MessageContent = ""
				msg.LastReason = "creation_limit_reached"
			}
			return nil
		})
		if updateErr != nil {
			return nil, updateErr
		}
		if msg.Terminal() {
			s.cleanupTerminal(msg)
		}
		if msg.State == types.StateCancelled {
			return nil, types.ErrMessageLimit
		}
		return msg, nil
	}
	return s.update(msgID, func(msg *types.ScheduledMessage) error {
		if msg.UserID != userID {
			return types.ErrNotOwner
		}
		if msg.State == types.StateCancelled {
			return types.ErrMessageFinal
		}
		if msg.State == types.StateCreating {
			msg.State = types.StatePending
		}
		return nil
	})
}

// DeleteScheduledMessage atomically cancels a job. Once publication starts,
// cancellation can no longer promise that the message will not appear.
func (s *kvStore) DeleteScheduledMessage(userID, msgID string) error {
	msg, err := s.update(msgID, func(msg *types.ScheduledMessage) error {
		if msg.UserID != userID {
			return types.ErrNotOwner
		}
		switch msg.DeliveryState() {
		case types.StateSending, types.StateReconciling, types.StateUncertain:
			return types.ErrMessageBusy
		case types.StateSent:
			return types.ErrMessageFinal
		case types.StateCancelled:
			return nil
		case types.StateCreating, types.StatePending, types.StateRetry, types.StateFailed, types.StateProcessing:
			msg.State = types.StateCancelled
			clearClaim(msg)
			msg.MessageContent = ""
			return nil
		default:
			return types.ErrMessageBusy
		}
	})
	if err != nil {
		return err
	}
	s.cleanupTerminal(msg)
	return nil
}

func (s *kvStore) GetScheduledMessage(msgID string) (*types.ScheduledMessage, error) {
	msg, _, err := s.load(msgID)
	return msg, err
}

func (s *kvStore) load(msgID string) (*types.ScheduledMessage, []byte, error) {
	var raw []byte
	if err := s.kv.Get(schedKey(msgID), &raw); err != nil {
		return nil, nil, err
	}
	if len(raw) == 0 {
		return nil, nil, types.ErrMessageNotFound
	}
	var msg types.ScheduledMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, nil, err
	}
	if msg.ID != msgID {
		return nil, nil, errors.New("invalid scheduled message record")
	}
	return &msg, raw, nil
}

func (s *kvStore) ListScheduledMessages() ([]*types.ScheduledMessage, error) {
	var messages []*types.ScheduledMessage
	// ListKeys filters after pagination. Iterate raw pages so index keys and
	// terminal receipts cannot hide pending jobs on a subsequent page.
	for page := 0; ; page++ {
		keys, err := s.kv.ListKeys(page, constants.MaxFetchScheduledMessages)
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			if !strings.HasPrefix(key, constants.SchedPrefix) {
				continue
			}
			msg, err := s.GetScheduledMessage(strings.TrimPrefix(key, constants.SchedPrefix))
			if err != nil {
				s.logger.Warn("Cannot read scheduled job", "key", key, "error_type", fmt.Sprintf("%T", err))
				continue
			}
			if msg.Terminal() {
				s.cleanupTerminal(msg)
				continue
			}
			if msg.State == types.StateCreating {
				msg, err = s.activateCreation(msg.UserID, msg.ID)
				if err != nil {
					s.logger.Warn("Cannot complete job creation", "key", key, "error_type", fmt.Sprintf("%T", err))
					continue
				}
				if msg.Terminal() {
					s.cleanupTerminal(msg)
					continue
				}
			}
			// Repair index entries lost to a partial creation or orphan cleanup.
			if err := s.modifyUserIndex(msg.UserID, msg.ID, indexRepair); err != nil {
				s.logger.Warn("Cannot repair job index", "message_id", msg.ID, "error_type", fmt.Sprintf("%T", err))
			}
			messages = append(messages, msg)
		}
		if len(keys) < constants.MaxFetchScheduledMessages {
			break
		}
	}
	return messages, nil
}

func (s *kvStore) ListUserMessageIDs(userID string) ([]string, error) {
	var ids []string
	err := s.kv.Get(indexKey(userID), &ids)
	return ids, err
}

func (s *kvStore) CleanupMessageFromUserIndex(userID, msgID string) error {
	return s.modifyUserIndex(userID, msgID, indexRemove)
}

func (s *kvStore) GenerateMessageID() string { return uuid.NewString() }

func (s *kvStore) ClaimScheduledMessage(userID, msgID string, now time.Time, manual bool) (*types.ScheduledMessage, error) {
	token := uuid.NewString()
	return s.update(msgID, func(msg *types.ScheduledMessage) error {
		if msg.UserID != userID {
			return types.ErrNotOwner
		}
		if msg.Terminal() {
			return types.ErrMessageFinal
		}
		state := msg.DeliveryState()
		switch state {
		case types.StateProcessing, types.StateSending, types.StateReconciling:
			if msg.LeaseUntil.After(now) {
				return types.ErrMessageBusy
			}
		case types.StatePending, types.StateRetry, types.StateUncertain:
			if !manual && !msg.Due(now) {
				return types.ErrNotDue
			}
		case types.StateFailed:
			if !manual {
				return types.ErrNotDue
			}
			msg.Attempts = 0 // Explicit Send starts a new bounded retry batch.
		default:
			return types.ErrMessageBusy
		}
		msg.ClaimToken = token
		msg.LeaseUntil = now.Add(types.DeliveryLease)
		msg.NextAttemptAt = time.Time{}
		if state == types.StateSending || state == types.StateReconciling || state == types.StateUncertain {
			msg.State = types.StateReconciling
		} else {
			msg.State = types.StateProcessing
		}
		return nil
	})
}

func (s *kvStore) StartDelivery(claim *types.ScheduledMessage, now time.Time) error {
	msg, err := s.update(claim.ID, func(msg *types.ScheduledMessage) error {
		if msg.UserID != claim.UserID || msg.ClaimToken != claim.ClaimToken || msg.ClaimToken == "" ||
			msg.DeliveryState() != types.StateProcessing || !msg.LeaseUntil.After(now) {
			return types.ErrLeaseLost
		}
		msg.State = types.StateSending
		msg.Attempts++
		return nil
	})
	if err == nil {
		*claim = *msg
	}
	return err
}

func (s *kvStore) FinishDelivery(claim *types.ScheduledMessage, result types.DeliveryResult, now time.Time) error {
	msg, err := s.update(claim.ID, func(msg *types.ScheduledMessage) error {
		if msg.UserID != claim.UserID || msg.ClaimToken != claim.ClaimToken || msg.ClaimToken == "" {
			return types.ErrLeaseLost
		}
		state := msg.DeliveryState()
		if state != types.StateSending && state != types.StateReconciling {
			return types.ErrLeaseLost
		}
		if state == types.StateReconciling && result.State != types.StateSent && result.State != types.StateUncertain {
			return errors.New("ambiguous publication cannot be retried")
		}
		switch result.State {
		case types.StateSent:
			if result.PostID == "" {
				return errors.New("missing publication receipt")
			}
			msg.PostID = result.PostID
			msg.MessageContent = ""
		case types.StateRetry, types.StateFailed, types.StateUncertain:
		default:
			return errors.New("invalid delivery result")
		}
		msg.State = result.State
		msg.LastReason = result.Reason
		clearClaim(msg)
		if msg.State == types.StateRetry {
			if msg.Attempts >= types.MaxDeliveryAttempts {
				msg.State = types.StateFailed
			} else {
				msg.NextAttemptAt = now.Add(types.RetryDelay(msg.Attempts))
			}
		}
		if msg.State == types.StateUncertain {
			msg.NextAttemptAt = now.Add(5 * time.Minute)
		}
		return nil
	})
	if err != nil {
		return err
	}
	*claim = *msg
	if msg.Terminal() {
		s.cleanupTerminal(msg)
	}
	return nil
}

func clearClaim(msg *types.ScheduledMessage) {
	msg.ClaimToken = ""
	msg.LeaseUntil = time.Time{}
	msg.NextAttemptAt = time.Time{}
}

func (s *kvStore) update(msgID string, change func(*types.ScheduledMessage) error) (*types.ScheduledMessage, error) {
	for attempt := 0; attempt < maxCASAttempts; attempt++ {
		msg, raw, err := s.load(msgID)
		if err != nil {
			return nil, err
		}
		if err := change(msg); err != nil {
			return nil, err
		}
		opts := []pluginapi.KVSetOption{pluginapi.SetAtomic(raw)}
		if msg.Terminal() {
			opts = append(opts, pluginapi.SetExpiry(types.ReceiptRetention))
		}
		written, err := s.kv.Set(schedKey(msgID), msg, opts...)
		if err != nil {
			return nil, err
		}
		if written {
			return msg, nil
		}
	}
	return nil, types.ErrConflict
}

func (s *kvStore) cleanupTerminal(msg *types.ScheduledMessage) {
	if err := s.modifyUserIndex(msg.UserID, msg.ID, indexRemove); err != nil {
		// The terminal receipt is committed. The next scan retries index cleanup.
		s.logger.Warn("Cannot clean up terminal job index", "message_id", msg.ID, "error_type", fmt.Sprintf("%T", err))
	}
}

func (s *kvStore) modifyUserIndex(userID, msgID string, change indexChange) error {
	for attempt := 0; attempt < maxCASAttempts; attempt++ {
		var raw []byte
		if err := s.kv.Get(indexKey(userID), &raw); err != nil {
			return err
		}
		var ids []string
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &ids); err != nil {
				return err
			}
		}
		idx := slices.Index(ids, msgID)
		if change != indexRemove {
			msg, err := s.GetScheduledMessage(msgID)
			if err != nil {
				return err
			}
			if msg.UserID != userID {
				return types.ErrNotOwner
			}
			if msg.Terminal() {
				return types.ErrMessageFinal
			}
			if idx >= 0 {
				return nil
			}
			if change == indexReserve && len(ids) >= s.maxUserMessages {
				return types.ErrMessageLimit
			}
			ids = append(ids, msgID)
		} else {
			if idx < 0 {
				return nil
			}
			// Revalidate AFTER reading the index snapshot. A concurrent insert
			// changes that snapshot and makes our removal CAS fail. Since creation
			// writes its canonical record first, a live reservation cannot vanish.
			msg, err := s.GetScheduledMessage(msgID)
			if err != nil && !errors.Is(err, types.ErrMessageNotFound) {
				return err
			}
			if err == nil && !msg.Terminal() {
				return nil
			}
			ids = slices.Delete(ids, idx, idx+1)
			if ids == nil {
				ids = []string{}
			}
		}
		written, err := s.kv.Set(indexKey(userID), ids, pluginapi.SetAtomic(raw))
		if err != nil {
			return err
		}
		if written {
			return nil
		}
	}
	return types.ErrConflict
}

func schedKey(id string) string     { return constants.SchedPrefix + id }
func indexKey(userID string) string { return constants.UserIndexPrefix + userID }

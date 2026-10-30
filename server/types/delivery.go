//revive:disable:var-naming // Package name is conventional for shared types.
package types

import (
	"errors"
	"time"
)

// DeliveryState is persisted with the job, independently of the user index.
type DeliveryState string

// Delivery states and limits define the persisted delivery protocol.
const (
	StateCreating       DeliveryState = "creating"
	StatePending        DeliveryState = "pending"
	StateProcessing     DeliveryState = "processing"
	StateSending        DeliveryState = "sending"
	StateReconciling    DeliveryState = "reconciling"
	StateRetry          DeliveryState = "retry"
	StateUncertain      DeliveryState = "uncertain"
	StateFailed         DeliveryState = "failed"
	StateSent           DeliveryState = "sent"
	StateCancelled      DeliveryState = "cancelled"
	MaxDeliveryAttempts               = 5
	DeliveryLease                     = 2 * time.Minute
	ReceiptRetention                  = 7 * 24 * time.Hour
	ScheduledJobProp                  = "poor_mans_scheduled_message_id"
)

// Delivery errors distinguish ownership, state, and concurrency conflicts.
var (
	ErrMessageNotFound   = errors.New("scheduled message not found")
	ErrMessageBusy       = errors.New("scheduled message is already being processed")
	ErrMessageFinal      = errors.New("scheduled message has already been sent or cancelled")
	ErrNotOwner          = errors.New("scheduled message belongs to another user")
	ErrLeaseLost         = errors.New("delivery claim has expired or was replaced")
	ErrNotDue            = errors.New("scheduled message is not due yet")
	ErrConflict          = errors.New("concurrent modification; please try again")
	ErrMessageLimit      = errors.New("scheduled message limit reached")
	ErrDeliveryUncertain = errors.New("delivery outcome is unknown; automatic re-publication is blocked while checking for the original post")
)

// DeliveryResult contains only receipt metadata and a controlled technical reason.
type DeliveryResult struct {
	State  DeliveryState
	PostID string
	Reason string
}

// DeliveryState treats records written before state tracking as pending jobs.
func (m *ScheduledMessage) DeliveryState() DeliveryState {
	if m.State == "" {
		return StatePending
	}
	return m.State
}

// Terminal jobs are receipts, never candidates for publication.
func (m *ScheduledMessage) Terminal() bool {
	return m.DeliveryState() == StateSent || m.DeliveryState() == StateCancelled
}

// Due includes expired claims and ambiguous attempts that need reconciliation.
func (m *ScheduledMessage) Due(now time.Time) bool {
	switch m.DeliveryState() {
	case StatePending:
		return !m.PostAt.After(now)
	case StateRetry, StateUncertain:
		return !m.NextAttemptAt.After(now)
	case StateProcessing, StateSending, StateReconciling:
		return !m.LeaseUntil.After(now)
	default:
		return false
	}
}

// RetryDelay provides bounded exponential backoff; attempts are capped separately.
func RetryDelay(attempt int) time.Duration {
	return time.Minute * time.Duration(1<<min(max(attempt-1, 0), 4))
}

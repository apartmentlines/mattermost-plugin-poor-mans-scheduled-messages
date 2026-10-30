package scheduler

import (
	"context"
	"errors"
	"fmt"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// SendNow uses the same atomic claim as scheduled delivery. Its argument is
// only an identity: content and state are reloaded from canonical storage.
func (s *Scheduler) SendNow(msg *types.ScheduledMessage) error { return s.send(msg, true) }

func (s *Scheduler) send(identity *types.ScheduledMessage, manual bool) error {
	select {
	case <-s.ctx.Done():
		return context.Canceled
	default:
	}
	if identity == nil {
		return types.ErrMessageNotFound
	}
	claim, err := s.store.ClaimScheduledMessage(identity.UserID, identity.ID, s.clock.Now().UTC(), manual)
	if err != nil {
		return err
	}
	if claim.State == types.StateReconciling {
		return s.reconcile(claim)
	}
	// This durable transition is the point after which cancellation is refused.
	// If it fails, no request is made and the expiring claim is safe to recover.
	if err := s.store.StartDelivery(claim, s.clock.Now().UTC()); err != nil {
		return err
	}
	post := &model.Post{
		UserId: claim.UserID, ChannelId: claim.ChannelID, RootId: claim.RootID,
		Message:       claim.MessageContent,
		PendingPostId: claim.UserID + "_" + claim.ID,
	}
	post.AddProp(types.ScheduledJobProp, claim.ID)
	postErr := s.poster.CreatePost(post)
	result := types.DeliveryResult{State: types.StateSent, PostID: post.Id}
	if postErr != nil {
		result = classifyFailure(postErr)
	}
	if postErr == nil && post.Id == "" {
		postErr = errors.New("publication returned no receipt")
		result = types.DeliveryResult{State: types.StateUncertain, Reason: "receipt_missing"}
	}
	if err := s.store.FinishDelivery(claim, result, s.clock.Now().UTC()); err != nil {
		// The sending record stays durable. Recovery only searches for the post;
		// it will never issue another blind CreatePost after this point.
		return fmt.Errorf("could not persist delivery result; the job is retained for recovery: %w", err)
	}
	if postErr != nil {
		s.logger.Warn("Scheduled delivery failed", "message_id", claim.ID, "state", claim.State, "reason", claim.LastReason)
		if claim.State == types.StateFailed || claim.State == types.StateUncertain {
			s.dmUserOnFailedMessage(claim, postErr)
		}
		return postErr
	}
	s.logger.Info("Scheduled message delivered", "message_id", claim.ID, "post_id", claim.PostID)
	return nil
}

// Only explicit rejections can safely be retried. Transport errors, timeouts,
// and server errors can occur after commit and must be reconciled instead.
func classifyFailure(err error) types.DeliveryResult {
	// pluginapi replaces HTTP 404 AppErrors with this sentinel, including
	// when the error reaches us through an additional wrapper.
	if errors.Is(err, pluginapi.ErrNotFound) {
		return types.DeliveryResult{State: types.StateFailed, Reason: "publication_rejected_404"}
	}
	var appErr *model.AppError
	if errors.As(err, &appErr) && appErr != nil {
		if appErr.StatusCode == 429 {
			return types.DeliveryResult{State: types.StateRetry, Reason: "rate_limited"}
		}
		if appErr.StatusCode >= 400 && appErr.StatusCode < 500 && appErr.StatusCode != 408 {
			return types.DeliveryResult{State: types.StateFailed, Reason: fmt.Sprintf("publication_rejected_%d", appErr.StatusCode)}
		}
	}
	return types.DeliveryResult{State: types.StateUncertain, Reason: "publication_outcome_unknown"}
}

func (s *Scheduler) reconcile(claim *types.ScheduledMessage) error {
	post, lookupErr := s.findPublishedPost(claim)
	result := types.DeliveryResult{State: types.StateUncertain, Reason: "publication_not_confirmed"}
	if lookupErr != nil {
		result.Reason = "reconciliation_failed"
	}
	if post != nil {
		result = types.DeliveryResult{State: types.StateSent, PostID: post.Id}
	}
	if err := s.store.FinishDelivery(claim, result, s.clock.Now().UTC()); err != nil {
		return err
	}
	if result.State == types.StateSent {
		return nil
	}
	return types.ErrDeliveryUncertain
}

func (s *Scheduler) findPublishedPost(msg *types.ScheduledMessage) (*model.Post, error) {
	// Bound a scan so a busy channel cannot monopolize the scheduler forever.
	// Hitting the bound leaves the job uncertain, never eligible for publication.
	const pageSize = 100
	for page := 0; page < 1000; page++ {
		posts, err := s.poster.GetPostsForChannel(msg.ChannelID, page, pageSize)
		if err != nil {
			return nil, err
		}
		if posts == nil {
			return nil, errors.New("empty post lookup response")
		}
		for _, post := range posts.Posts {
			if post != nil && post.Id != "" && post.UserId == msg.UserID && post.ChannelId == msg.ChannelID && post.RootId == msg.RootID && post.GetProp(types.ScheduledJobProp) == msg.ID {
				return post, nil
			}
		}
		if len(posts.Order) < pageSize {
			return nil, nil
		}
	}
	return nil, errors.New("post reconciliation scan limit reached")
}

func (s *Scheduler) dmUserOnFailedMessage(msg *types.ScheduledMessage, postErr error) {
	r := formatter.ForUser(s.presentation, msg.UserID)
	channelInfo := s.linker.MakeChannelLink(s.linker.GetInfoOrUnknown(msg.ChannelID), r.DisplaySettings)
	message := r.FormatSchedulerFailure(channelInfo, postErr, msg.MessageContent)
	if err := s.poster.DM(s.botID, msg.UserID, &model.Post{Message: message}); err != nil {
		s.logger.Error("Cannot notify author of delivery failure", "message_id", msg.ID, "error_type", fmt.Sprintf("%T", err))
	}
}

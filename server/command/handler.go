// --- File: pkg/command/handler.go ---
package command

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/scheduler"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
)

type Handler struct {
	api      *pluginapi.Client
	store    store.Store
	scheduler *scheduler.Scheduler
}

func NewHandler(api *pluginapi.Client, store store.Store, sched *scheduler.Scheduler) *Handler {
	return &Handler{
		api: api,
		store: store,
		scheduler: sched,
	}
}

func (h *Handler) Definition() *model.Command {
	return &model.Command{
		Trigger:          "schedule",
		AutoComplete:     true,
		AutoCompleteDesc: "Schedule messages to be sent later",
		AutoCompleteHint: "at <time> [on <date>] message <text> | list | delete <id>",
		DisplayName:      "Schedule",
		Description:      "Send messages at a future time.",
	}
}

func (h *Handler) Execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	ctx := context.Background()
	text := strings.TrimSpace(args.Command[len("/schedule"):])

	switch {
	case strings.HasPrefix(text, "list"):
		return h.BuildEphemeralList(ctx, args), nil
	case strings.HasPrefix(text, "delete"):
		return h.handleDelete(ctx, args, text), nil
	default:
		return h.handleSchedule(ctx, args, text), nil
	}
}

func (h *Handler) BuildEphemeralList(ctx context.Context, args *model.CommandArgs) *model.CommandResponse {
	ids, err := h.store.ListUserMessageIDs(ctx, args.UserId)
	if err != nil || len(ids) == 0 {
		return h.ephemeral("You have no scheduled messages.")
	}

	var msgs []*types.ScheduledMessage
	for _, id := range ids {
		msg, err := h.store.GetScheduledMessage(ctx, id)
			if err == nil {
				msgs = append(msgs, msg)
			}
	}
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].PostAt.Before(msgs[j].PostAt)
	})

	var attachments []*model.SlackAttachment
	for _, m := range msgs {
		loc, _ := time.LoadLocation(m.Timezone)
		localTime := m.PostAt.In(loc)
		header := fmt.Sprintf("### %s\n%s",
			localTime.Format("2006-01-02 15:04"), m.MessageContent)
		attachments = append(attachments, createAttachment(header, m.ID))
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         "### Scheduled Messages",
		Props: map[string]interface{}{
			"attachments": attachments,
		},
	}
}

func createAttachment(text string, messageID string) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text: text,
		Actions: []*model.PostAction{
			{
				Id:    "delete",
				Name:  "Delete",
				Style: "danger",
				Integration: &model.PostActionIntegration{
					URL: "/plugins/com.mattermost.plugin-poor-mans-scheduled-messages/api/v1/delete",
					Context: map[string]interface{}{
						"action": "delete",
						"id":     messageID,
					},
				},
			},
		},
	}
}

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	p.logger.Debug("Received HTTP request", "method", r.Method, "url", r.URL.String())
	router := mux.NewRouter()
	router.Use(p.MattermostAuthorizationRequired)
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/delete", p.UserDeleteMessage).Methods(http.MethodPost)
	router.ServeHTTP(w, r)
}

func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get(constants.HTTPHeaderMattermostUserID)
		p.logger.Debug("Checking Mattermost authorization", "user_id", userID, "url", r.URL.String())
		if userID == "" {
			p.logger.Warn("Authorization failed: Missing user ID header", "header", constants.HTTPHeaderMattermostUserID, "remote_addr", r.RemoteAddr)
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}
		p.logger.Debug("Authorization successful", "user_id", userID)
		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) UserDeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get(constants.HTTPHeaderMattermostUserID)
	p.logger.Debug("Handling UserDeleteMessage request", "user_id", userID)

	p.logger.Debug("Parsing delete request body", "user_id", userID)
	req, msgID, err := parseDeleteRequest(p, r)
	if err != nil {
		p.logger.Error("Failed to parse delete request", "user_id", userID, "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	p.logger.Debug("Successfully parsed delete request", "user_id", userID, "message_id", msgID, "post_id", req.PostId, "channel_id", req.ChannelId)

	p.logger.Debug("Calling command layer UserDeleteMessage", "user_id", userID, "message_id", msgID)
	deletedMsg, err := p.Command.UserDeleteMessage(userID, msgID)
	args := &model.CommandArgs{
		UserId: userID,
	}
	p.logger.Debug("Building updated ephemeral list", "user_id", userID)
	updatedList := p.Command.BuildEphemeralList(args)
	p.updateEphemeralPostWithList(userID, req.PostId, req.ChannelId, updatedList)
	if err != nil {
		p.logger.Error("Command layer failed to delete message", "user_id", userID, "message_id", msgID, "error", err)
		http.Error(w, fmt.Sprintf("Failed to delete message: %v", err), http.StatusInternalServerError)
		p.sendDeletionError(userID, req.ChannelId, msgID, err)
		return
	}
	p.logger.Info("Successfully deleted message via command layer", "user_id", userID, "message_id", msgID)
	p.sendDeletionConfirmation(userID, req.ChannelId, deletedMsg)

	p.logger.Debug("UserDeleteMessage request completed successfully", "user_id", userID, "message_id", msgID)
}

func (p *Plugin) buildEphemeralListUpdate(userID, postID, channelID string, updatedList *model.CommandResponse) *model.Post {
	p.logger.Debug("Building ephemeral post update structure", "user_id", userID, "post_id", postID, "channel_id", channelID)
	post := &model.Post{
		Id:        postID,
		UserId:    userID,
		ChannelId: channelID,
		Props: map[string]any{
			"attachments": updatedList.Props["attachments"],
		},
	}
	attachmentsSlice := updatedList.Props["attachments"].([]*model.SlackAttachment)
	if len(attachmentsSlice) == 0 {
		p.logger.Debug("Attachments slice is empty, setting EmptyListMessage", "user_id", userID, "post_id", postID)
		post.Message = constants.EmptyListMessage
	}
	return post
}

func parseDeleteRequest(p *Plugin, r *http.Request) (*model.PostActionIntegrationRequest, string, error) {
	p.logger.Debug("Decoding JSON body for delete request")
	var req model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		p.logger.Error("Failed to decode JSON body", "error", err)
		return nil, "", fmt.Errorf("invalid request body: %w", err)
	}

	p.logger.Debug("Validating delete request context", "context", req.Context)
	action, actionOk := req.Context["action"].(string)
	msgID, idOk := req.Context["id"].(string)

	if !actionOk || action != "delete" || !idOk || msgID == "" {
		err := errors.New("invalid delete request context: missing or invalid action/id")
		p.logger.Error("Delete request context validation failed", "error", err, "action", action, "action_ok", actionOk, "msg_id", msgID, "id_ok", idOk)
		return nil, "", err
	}

	p.logger.Debug("Delete request parsed and validated successfully", "action", action, "message_id", msgID)
	return &req, msgID, nil
}

func (p *Plugin) updateEphemeralPostWithList(userID string, postID string, channelID string, updatedList *model.CommandResponse) {
	p.logger.Debug("Preparing to update ephemeral post with new list", "user_id", userID, "post_id", postID, "channel_id", channelID)
	updatedPost := p.buildEphemeralListUpdate(userID, postID, channelID, updatedList)
	p.poster.UpdateEphemeralPost(userID, updatedPost)
	p.logger.Debug("Successfully requested ephemeral post update", "user_id", userID, "post_id", postID, "channel_id", channelID)
}

func (p *Plugin) sendDeletionConfirmation(userID string, channelID string, deletedMsg *types.ScheduledMessage) {
	p.logger.Debug("Preparing deletion confirmation message", "user_id", userID, "channel_id", channelID, "message_id", deletedMsg.ID, "timezone", deletedMsg.Timezone)
	loc, err := time.LoadLocation(deletedMsg.Timezone)
	if err != nil {
		p.logger.Warn("Failed to load timezone for confirmation message, falling back to UTC", "user_id", userID, "message_id", deletedMsg.ID, "timezone", deletedMsg.Timezone, "error", err)
		loc = time.UTC
	}
	humanTime := deletedMsg.PostAt.In(loc).Format(constants.TimeLayout)
	p.logger.Debug("Formatted time for confirmation message", "user_id", userID, "message_id", deletedMsg.ID, "formatted_time", humanTime, "location", loc.String())
	channelInfo := p.Channel.MakeChannelLink(p.Channel.GetInfoOrUnknown(deletedMsg.ChannelID))
	confirmation := &model.Post{
		UserId:    userID,
		ChannelId: channelID,
		Message:   fmt.Sprintf("%s Message scheduled for **%s** %s has been deleted.", constants.EmojiSuccess, humanTime, channelInfo),
	}
	p.logger.Debug("Sending ephemeral deletion confirmation post", "user_id", userID, "channel_id", channelID, "message_id", deletedMsg.ID)
	p.poster.SendEphemeralPost(userID, confirmation)
	p.logger.Debug("Successfully sent ephemeral deletion confirmation", "user_id", userID, "channel_id", channelID, "message_id", deletedMsg.ID)
}

func (p *Plugin) sendDeletionError(userID string, channelID string, msgID string, err error) {
	p.logger.Debug("Preparing deletion error message", "user_id", userID, "channel_id", channelID, "message_id", msgID, "error", err)
	alert := &model.Post{
		UserId:    userID,
		ChannelId: channelID,
		Message:   fmt.Sprintf("%s Could not delete message: %v", constants.EmojiError, err),
	}
	p.logger.Debug("Sending ephemeral deletion confirmation post", "user_id", userID, "channel_id", channelID, "message_id", msgID)
	p.poster.SendEphemeralPost(userID, alert)
	p.logger.Debug("Successfully sent ephemeral deletion confirmation", "user_id", userID, "channel_id", channelID, "message_id", msgID)
}

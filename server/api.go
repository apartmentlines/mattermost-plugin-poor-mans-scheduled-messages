package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()
	router.Use(p.MattermostAuthorizationRequired)
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/delete", p.UserDeleteMessage).Methods(http.MethodPost)
	router.ServeHTTP(w, r)
}

func (p *Plugin) MattermostAuthorizationRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("Mattermost-User-ID")
		if userID == "" {
			http.Error(w, "Not authorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p *Plugin) UserDeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	p.client.Log.Debug("Received request to delete message", "user_id", userID)
	req, msgID, err := parseDeleteRequest(r)
	if err != nil {
		p.client.Log.Warn("Failed to parse delete request", "error", err, "user_id", userID)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	deletedMsg, err := p.Command.UserDeleteMessage(userID, msgID)
	if err != nil {
		p.client.Log.Error("Failed to delete message", "error", err, "user_id", userID, "message_id", msgID)
		http.Error(w, fmt.Sprintf("Failed to delete message: %v", err), http.StatusInternalServerError)
		return
	}
	p.client.Log.Debug("Deleted message", "user_id", userID, "message_id", msgID)
	args := &model.CommandArgs{
		UserId: userID,
	}
	updatedList := p.Command.BuildEphemeralList(args)
	p.updateEphemeralPostWithList(userID, req.PostId, req.ChannelId, updatedList)
	p.sendDeletionConfirmation(userID, req.ChannelId, deletedMsg)
}

func parseDeleteRequest(r *http.Request) (*model.PostActionIntegrationRequest, string, error) {
	var req model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, "", fmt.Errorf("invalid request body: %w", err)
	}
	action, _ := req.Context["action"].(string)
	msgID, _ := req.Context["id"].(string)
	if action != "delete" || msgID == "" {
		return nil, "", errors.New("invalid delete request context: missing or invalid action/id")
	}
	return &req, msgID, nil
}

func (p *Plugin) updateEphemeralPostWithList(userID string, postID string, channelID string, updatedList *model.CommandResponse) {
	updatedPost := &model.Post{
		Id:        postID,
		UserId:    userID,
		ChannelId: channelID,
		Props: map[string]any{
			"attachments": updatedList.Props["attachments"],
		},
	}
	p.client.Post.UpdateEphemeralPost(userID, updatedPost)
	p.client.Log.Debug("Updated ephemeral post with current scheduled task list", "user_id", userID, "post_id", postID)
}

func (p *Plugin) sendDeletionConfirmation(userID string, channelID string, deletedMsg *types.ScheduledMessage) {
	loc, err := time.LoadLocation(deletedMsg.Timezone)
	if err != nil {
		p.client.Log.Warn("Failed to load timezone for confirmation", "timezone", deletedMsg.Timezone, "error", err)
		loc = time.UTC
	}
	humanTime := deletedMsg.PostAt.In(loc).Format("Jan 2, 2006 3:04 PM")
	confirmation := &model.Post{
		UserId:    userID,
		ChannelId: channelID,
		Message:   fmt.Sprintf("✅ Message scheduled for **%s** has been deleted.", humanTime),
	}
	p.client.Post.SendEphemeralPost(userID, confirmation)
	p.client.Log.Debug("Sent deletion confirmation", "user_id", userID, "channel_id", channelID)
}

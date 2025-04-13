package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

	var req model.PostActionIntegrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	action := req.Context["action"]
	msgID, ok := req.Context["id"].(string)
	if action != "delete" || !ok || msgID == "" {
		http.Error(w, "Invalid delete request", http.StatusBadRequest)
		return
	}

	msg, getErr := p.Store.GetScheduledMessage(msgID)
	if getErr != nil || msg.UserID != userID {
		http.Error(w, "Not found or not authorized", http.StatusForbidden)
		return
	}

	deleteErr := p.Store.DeleteScheduledMessage(msg.UserID, msgID)
	if deleteErr != nil {
		http.Error(w, deleteErr.Error(), http.StatusInternalServerError)
		return
	}

	args := &model.CommandArgs{
		UserId:    userID,
		ChannelId: msg.ChannelID,
	}
	updatedList := p.Command.BuildEphemeralList(args)

	updatedPost := &model.Post{
		Id:        req.PostId,
		UserId:    userID,
		ChannelId: req.ChannelId,
		Props: map[string]interface{}{
			"attachments": updatedList.Props["attachments"],
		},
	}

	p.client.Post.UpdateEphemeralPost(userID, updatedPost)

	// Format the original post time for the deletion confirmation
	loc, _ := time.LoadLocation(msg.Timezone)
	humanTime := msg.PostAt.In(loc).Format("Jan 2, 2006 3:04 PM")

	// Send new ephemeral confirmation
	confirmation := &model.Post{
		UserId:    userID,
		ChannelId: req.ChannelId,
		Message:   fmt.Sprintf("✅ Message scheduled for **%s** has been deleted.", humanTime),
	}

	p.client.Post.SendEphemeralPost(userID, confirmation)
}

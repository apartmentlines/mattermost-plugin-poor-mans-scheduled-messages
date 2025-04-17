package main

import (
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

func EnsureBot(api *pluginapi.Client) (string, error) {
	bot := &model.Bot{
		Username:    "poor-mans-scheduled-messages",
		DisplayName: "Message Scheduler",
		Description: "Poor Man's Scheduled Messages Bot",
	}
	profileImage := pluginapi.ProfileImagePath("assets/profile.png")
	botUserID, err := api.Bot.EnsureBot(bot, profileImage)
	if err != nil {
		return "", fmt.Errorf("failed to ensure bot: %w", err)
	}
	return botUserID, nil
}

package bot

import (
	"fmt"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/mattermost/mattermost/server/public/model"
)

func EnsureBot(botAPI ports.BotService, imgSvc ports.BotProfileImageService) (string, error) {
	bot := &model.Bot{
		Username:    "poor-mans-scheduled-messages",
		DisplayName: "Message Scheduler",
		Description: "Poor Man's Scheduled Messages Bot",
	}
	profileImage := imgSvc.ProfileImagePath("assets/profile.png")
	botUserID, err := botAPI.EnsureBot(bot, profileImage)
	if err != nil {
		return "", fmt.Errorf("failed to ensure bot: %w", err)
	}
	return botUserID, nil
}

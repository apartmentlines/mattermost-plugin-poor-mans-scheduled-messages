package bot

import (
	"fmt"
	"path/filepath"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/mattermost/mattermost/server/public/model"
)

func EnsureBot(botAPI ports.BotService, imgSvc ports.BotProfileImageService) (string, error) {
	bot := &model.Bot{
		Username:    "poor-mans-scheduled-messages",
		DisplayName: "Message Scheduler",
		Description: "Poor Man's Scheduled Messages Bot",
	}
	profileImagePath := filepath.Join(constants.AssetsDir, constants.ProfileImageFilename)
	profileImage := imgSvc.ProfileImagePath(profileImagePath)
	botUserID, err := botAPI.EnsureBot(bot, profileImage)
	if err != nil {
		return "", fmt.Errorf("failed to ensure bot: %w", err)
	}
	return botUserID, nil
}

// Package mm provides Mattermost adapter helpers.
package mm

import (
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type botProfileImageServiceWrapper struct{}

// NewBotProfileImageService returns a BotProfileImageService backed by pluginapi helpers.
func NewBotProfileImageService() ports.BotProfileImageService {
	return botProfileImageServiceWrapper{}
}

func (botProfileImageServiceWrapper) ProfileImagePath(p string) pluginapi.EnsureBotOption {
	return pluginapi.ProfileImagePath(p)
}

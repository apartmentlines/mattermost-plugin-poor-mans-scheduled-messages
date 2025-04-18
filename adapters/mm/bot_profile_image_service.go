package mm

import "github.com/mattermost/mattermost/server/public/pluginapi"

type BotProfileImageServiceWrapper struct{}

func (BotProfileImageServiceWrapper) ProfileImagePath(p string) pluginapi.EnsureBotOption {
	return pluginapi.ProfileImagePath(p)
}

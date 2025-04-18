package mm

import "github.com/mattermost/mattermost/server/public/pluginapi"

type ListMatchingService struct{}

func (ListMatchingService) WithPrefix(p string) pluginapi.ListKeysOption {
	return pluginapi.WithPrefix(p)
}

package channel

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Channel struct {
	api *pluginapi.Client
}

func New(api *pluginapi.Client) *Channel {
	return &Channel{
		api: api,
	}
}

type Info struct {
	ChannelID   string
	ChannelType model.ChannelType
	ChannelLink string
	TeamName    string
}

func (c *Channel) GetInfo(channelID string) (*Info, error) {
	channel, channelGetErr := c.api.Channel.Get(channelID)
	if channelGetErr != nil {
		return nil, fmt.Errorf("failed to get channel %s: %w", channelID, channelGetErr)
	}
	if channel.Type == model.ChannelTypeDirect || channel.Type == model.ChannelTypeGroup {
		members, listMembersErr := c.api.Channel.ListMembers(channel.Id, 0, 100)
		if listMembersErr != nil {
			return nil, fmt.Errorf("failed to get members of channel %s: %w", channel.Id, listMembersErr)
		}
		usernames, err := c.mapMembersToUsernames(members)
		if err != nil {
			return nil, err
		}
		dmGroupName := strings.Join(usernames, ", ")
		return &Info{
			ChannelID:   channel.Id,
			ChannelType: channel.Type,
			ChannelLink: dmGroupName,
		}, nil
	}
	team, teamGetErr := c.api.Team.Get(channel.TeamId)
	if teamGetErr != nil {
		return nil, fmt.Errorf("failed to get team %s: %w", channel.TeamId, teamGetErr)
	}
	return &Info{
		ChannelID:   channel.Id,
		ChannelType: channel.Type,
		ChannelLink: fmt.Sprintf("~%s", channel.Name),
		TeamName:    team.DisplayName,
	}, nil
}

func (c *Channel) UnknownChannel() *Info {
	return &Info{
		ChannelLink: "N/A",
	}
}

func (c *Channel) GetInfoOrUnknown(channelID string) *Info {
	channelInfo, getChannelErr := c.GetInfo(channelID)
	if getChannelErr == nil {
		return channelInfo
	}
	c.api.Log.Error("Failed to get channel info", "channel_id", channelID, "error", getChannelErr)
	return c.UnknownChannel()
}

func (c *Channel) MakeChannelLink(channelInfo *Info) string {
	if channelInfo.ChannelID == "" {
		return channelInfo.ChannelLink
	}
	if channelInfo.ChannelType == model.ChannelTypeDirect || channelInfo.ChannelType == model.ChannelTypeGroup {
		return fmt.Sprintf("in direct message with: %s", channelInfo.ChannelLink)
	}
	return fmt.Sprintf("in channel: %s", channelInfo.ChannelLink)
}

func (c *Channel) mapMembersToUsernames(members []*model.ChannelMember) ([]string, error) {
	var usernames []string
	for _, member := range members {
		user, err := c.api.User.Get(member.UserId)
		if err != nil {
			return nil, fmt.Errorf("failed to get user %s: %w", member.UserId, err)
		}
		usernames = append(usernames, "@"+user.Username)
	}
	return usernames, nil
}

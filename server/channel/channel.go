package channel

import (
	"fmt"
	"strings"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/mattermost/mattermost/server/public/model"
)

type Channel struct {
	logger     ports.Logger
	channelAPI ports.ChannelDataService
	teamAPI    ports.TeamService
	userAPI    ports.UserService
}

func New(
	logger ports.Logger,
	channelAPI ports.ChannelDataService,
	teamAPI ports.TeamService,
	userAPI ports.UserService,
) *Channel {
	return &Channel{
		logger:     logger,
		channelAPI: channelAPI,
		teamAPI:    teamAPI,
		userAPI:    userAPI,
	}
}

func (c *Channel) GetInfo(channelID string) (*ports.ChannelInfo, error) {
	channel, channelGetErr := c.channelAPI.Get(channelID)
	if channelGetErr != nil {
		return nil, fmt.Errorf("failed to get channel %s: %w", channelID, channelGetErr)
	}
	if channel.Type == model.ChannelTypeDirect || channel.Type == model.ChannelTypeGroup {
		members, listMembersErr := c.channelAPI.ListMembers(channel.Id, 0, 100)
		if listMembersErr != nil {
			return nil, fmt.Errorf("failed to get members of channel %s: %w", channel.Id, listMembersErr)
		}
		usernames, err := c.mapMembersToUsernames(members)
		if err != nil {
			return nil, err
		}
		dmGroupName := strings.Join(usernames, ", ")
		return &ports.ChannelInfo{
			ChannelID:   channel.Id,
			ChannelType: channel.Type,
			ChannelLink: dmGroupName,
		}, nil
	}
	team, teamGetErr := c.teamAPI.Get(channel.TeamId)
	if teamGetErr != nil {
		return nil, fmt.Errorf("failed to get team %s: %w", channel.TeamId, teamGetErr)
	}
	return &ports.ChannelInfo{
		ChannelID:   channel.Id,
		ChannelType: channel.Type,
		ChannelLink: fmt.Sprintf("~%s", channel.Name),
		TeamName:    team.DisplayName,
	}, nil
}

func (c *Channel) UnknownChannel() *ports.ChannelInfo {
	return &ports.ChannelInfo{
		ChannelLink: "N/A",
	}
}

func (c *Channel) GetInfoOrUnknown(channelID string) *ports.ChannelInfo {
	channelInfo, getChannelErr := c.GetInfo(channelID)
	if getChannelErr == nil {
		return channelInfo
	}
	c.logger.Error("Failed to get channel info", "channel_id", channelID, "error", getChannelErr)
	return c.UnknownChannel()
}

func (c *Channel) MakeChannelLink(channelInfo *ports.ChannelInfo) string {
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
		user, err := c.userAPI.Get(member.UserId)
		if err != nil {
			return nil, fmt.Errorf("failed to get user %s: %w", member.UserId, err)
		}
		usernames = append(usernames, "@"+user.Username)
	}
	return usernames, nil
}

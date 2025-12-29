// Package channel provides channel metadata helpers.
package channel

import (
	"fmt"
	"strings"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/mattermost/mattermost/server/public/model"
)

// Channel provides channel lookup and formatting helpers.
type Channel struct {
	logger     ports.Logger
	channelAPI ports.ChannelDataService
	teamAPI    ports.TeamService
	userAPI    ports.UserService
}

// New constructs a Channel service.
func New(
	logger ports.Logger,
	channelAPI ports.ChannelDataService,
	teamAPI ports.TeamService,
	userAPI ports.UserService,
) *Channel {
	logger.Debug("Creating new Channel service")
	return &Channel{
		logger:     logger,
		channelAPI: channelAPI,
		teamAPI:    teamAPI,
		userAPI:    userAPI,
	}
}

// GetInfo fetches channel metadata for a given channel ID.
func (c *Channel) GetInfo(channelID string) (*ports.ChannelInfo, error) {
	c.logger.Debug("Getting channel info", "channel_id", channelID)
	channel, channelGetErr := c.channelAPI.Get(channelID)
	if channelGetErr != nil {
		c.logger.Error("Failed to get channel from API", "channel_id", channelID, "error", channelGetErr)
		return nil, fmt.Errorf("failed to get channel %s: %w", channelID, channelGetErr)
	}

	c.logger.Debug("Determining channel type", "channel_id", channelID, "channel_type", channel.Type)
	switch channel.Type {
	case model.ChannelTypeDirect, model.ChannelTypeGroup:
		return c.getDirectOrGroupChannelInfo(channel)
	default:
		return c.getPublicOrPrivateChannelInfo(channel)
	}
}

func (c *Channel) getDirectOrGroupChannelInfo(channel *model.Channel) (*ports.ChannelInfo, error) {
	c.logger.Debug("Getting direct or group channel info", "channel_id", channel.Id)
	members, listMembersErr := c.channelAPI.ListMembers(channel.Id, constants.DefaultPage, constants.DefaultChannelMembersPerPage)
	if listMembersErr != nil {
		c.logger.Error("Failed to list channel members", "channel_id", channel.Id, "error", listMembersErr)
		return nil, fmt.Errorf("failed to get members of channel %s: %w", channel.Id, listMembersErr)
	}
	usernames, err := c.mapMembersToUsernames(members)
	if err != nil {
		// Error already logged in mapMembersToUsernames
		return nil, err
	}
	dmGroupName := strings.Join(usernames, ", ")
	c.logger.Debug("Successfully retrieved direct/group channel info", "channel_id", channel.Id, "member_usernames", dmGroupName)
	return &ports.ChannelInfo{
		ChannelID:   channel.Id,
		ChannelType: channel.Type,
		ChannelLink: dmGroupName,
	}, nil
}

func (c *Channel) getPublicOrPrivateChannelInfo(channel *model.Channel) (*ports.ChannelInfo, error) {
	c.logger.Debug("Getting public or private channel info", "channel_id", channel.Id, "team_id", channel.TeamId)
	team, err := c.teamAPI.Get(channel.TeamId)
	if err != nil {
		c.logger.Error("Failed to get team info", "team_id", channel.TeamId, "error", err)
		return nil, fmt.Errorf("failed to get team %s: %w", channel.TeamId, err)
	}
	c.logger.Debug("Successfully retrieved public/private channel info", "channel_id", channel.Id, "team_name", team.DisplayName, "channel_name", channel.Name)
	return &ports.ChannelInfo{
		ChannelID:   channel.Id,
		ChannelType: channel.Type,
		ChannelLink: fmt.Sprintf("~%s", channel.Name),
		TeamName:    team.DisplayName,
	}, nil
}

// UnknownChannel returns placeholder channel info for unknown channels.
func (c *Channel) UnknownChannel() *ports.ChannelInfo {
	c.logger.Debug("Returning unknown channel info placeholder")
	return &ports.ChannelInfo{
		ChannelLink: constants.UnknownChannelPlaceholder,
	}
}

// GetInfoOrUnknown returns channel info or a placeholder on error.
func (c *Channel) GetInfoOrUnknown(channelID string) *ports.ChannelInfo {
	c.logger.Debug("Getting channel info or returning unknown placeholder", "channel_id", channelID)
	channelInfo, getChannelErr := c.GetInfo(channelID)
	if getChannelErr == nil {
		c.logger.Debug("Successfully got channel info", "channel_id", channelID)
		return channelInfo
	}
	c.logger.Warn("GetInfo failed, returning unknown channel info", "channel_id", channelID, "error", getChannelErr)
	return c.UnknownChannel()
}

// MakeChannelLink formats a channel link string for display.
func (c *Channel) MakeChannelLink(channelInfo *ports.ChannelInfo) string {
	c.logger.Debug("Making channel link string", "channel_id", channelInfo.ChannelID, "channel_type", channelInfo.ChannelType, "channel_link_raw", channelInfo.ChannelLink)
	if channelInfo.ChannelID == "" {
		c.logger.Debug("Channel ID is empty, returning raw link (unknown channel)", "channel_link", channelInfo.ChannelLink)
		return channelInfo.ChannelLink
	}
	if channelInfo.ChannelType == model.ChannelTypeDirect || channelInfo.ChannelType == model.ChannelTypeGroup {
		return fmt.Sprintf("in direct message with: %s", channelInfo.ChannelLink)
	}
	return fmt.Sprintf("in channel: %s", channelInfo.ChannelLink)
}

func (c *Channel) mapMembersToUsernames(members []*model.ChannelMember) ([]string, error) {
	c.logger.Debug("Mapping channel members to usernames", "member_count", len(members))
	var usernames []string
	for _, member := range members {
		c.logger.Debug("Getting user info for member", "user_id", member.UserId)
		user, err := c.userAPI.Get(member.UserId)
		if err != nil {
			c.logger.Error("Failed to get user info", "user_id", member.UserId, "error", err)
			return nil, fmt.Errorf("failed to get user %s: %w", member.UserId, err)
		}
		usernames = append(usernames, "@"+user.Username)
	}
	c.logger.Debug("Successfully mapped members to usernames", "usernames", usernames)
	return usernames, nil
}

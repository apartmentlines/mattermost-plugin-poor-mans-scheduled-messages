package i18n

import "fmt"

// ChannelLinkDirect formats the scheduled-message context for a 1:1 direct message.
func ChannelLinkDirect(locale, usernames string) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("у прямому повідомленні з: %s", usernames)
	case "ru":
		return fmt.Sprintf("в личном сообщении с: %s", usernames)
	default:
		return fmt.Sprintf("in direct message with: %s", usernames)
	}
}

// ChannelLinkGroup formats the context for a group message channel.
func ChannelLinkGroup(locale, usernames string) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("у груповому повідомленні з: %s", usernames)
	case "ru":
		return fmt.Sprintf("в групповом сообщении с: %s", usernames)
	default:
		return fmt.Sprintf("in group message with: %s", usernames)
	}
}

// ChannelLinkInChannel formats the context for a team channel (open or private).
func ChannelLinkInChannel(locale, channelRef string) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("у каналі: %s", channelRef)
	case "ru":
		return fmt.Sprintf("в канале: %s", channelRef)
	default:
		return fmt.Sprintf("in channel: %s", channelRef)
	}
}

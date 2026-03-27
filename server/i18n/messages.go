package i18n

import "fmt"

// ScheduleSuccess confirms a newly scheduled message (composer or slash command).
func ScheduleSuccess(locale, emoji, timeStr, tzIANA, channelLink string) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("%s Заплановано повідомлення на %s (%s) %s", emoji, timeStr, tzIANA, channelLink)
	case "ru":
		return fmt.Sprintf("%s Запланировано сообщение на %s (%s) %s", emoji, timeStr, tzIANA, channelLink)
	default:
		return fmt.Sprintf("%s Scheduled message for %s (%s) %s", emoji, timeStr, tzIANA, channelLink)
	}
}

// ScheduledMessageDeleted confirms removal from the scheduled list UI.
func ScheduledMessageDeleted(locale, emoji, boldTime, channelInfo string) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("%s Повідомлення, заплановане на **%s** %s, видалено.", emoji, boldTime, channelInfo)
	case "ru":
		return fmt.Sprintf("%s Сообщение, запланированное на **%s** %s, удалено.", emoji, boldTime, channelInfo)
	default:
		return fmt.Sprintf("%s Message scheduled for **%s** %s has been deleted.", emoji, boldTime, channelInfo)
	}
}

// ScheduledMessageSent confirms a message was sent immediately from the list UI.
func ScheduledMessageSent(locale, emoji, boldTime, channelInfo string) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("%s Повідомлення, заплановане на **%s** %s, надіслано.", emoji, boldTime, channelInfo)
	case "ru":
		return fmt.Sprintf("%s Сообщение, запланированное на **%s** %s, отправлено.", emoji, boldTime, channelInfo)
	default:
		return fmt.Sprintf("%s Message scheduled for **%s** %s has been sent.", emoji, boldTime, channelInfo)
	}
}

// SchedulerFailure is sent as a DM when a scheduled post fails to publish.
func SchedulerFailure(locale, emoji, channelCtx string, postErr error, originalMsg string) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("%s Не вдалося надіслати заплановане повідомлення %s: %v — початковий текст: %s", emoji, channelCtx, postErr, originalMsg)
	case "ru":
		return fmt.Sprintf("%s Не удалось отправить запланированное сообщение %s: %v — исходный текст: %s", emoji, channelCtx, postErr, originalMsg)
	default:
		return fmt.Sprintf("%s Error scheduling message %s: %v -- original message: %s", emoji, channelCtx, postErr, originalMsg)
	}
}

// SchedulePersistError is shown when saving to the plugin store fails after validation.
func SchedulePersistError(locale, emoji, timeStr, tzIANA, channelCtx string, err error) string {
	switch NormalizeLocale(locale) {
	case "uk":
		return fmt.Sprintf("%s Помилка планування повідомлення на %s (%s) %s:  %v", emoji, timeStr, tzIANA, channelCtx, err)
	case "ru":
		return fmt.Sprintf("%s Ошибка планирования сообщения на %s (%s) %s:  %v", emoji, timeStr, tzIANA, channelCtx, err)
	default:
		return fmt.Sprintf("%s Error scheduling message for %s (%s) %s:  %v", emoji, timeStr, tzIANA, channelCtx, err)
	}
}

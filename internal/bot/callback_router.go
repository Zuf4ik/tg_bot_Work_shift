package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	if callback == nil || callback.Message == nil || callback.Message.Chat == nil || callback.From == nil {
		return
	}

	if callback.Message.Chat.Type != "private" || callback.From.ID != callback.Message.Chat.ID {
		b.answerCallback(callback.ID, "Нет доступа")
		return
	}

	data := callback.Data

	if strings.HasPrefix(data, "calendar:") {
		b.handleShiftCalendarCallback(callback)
		return
	}

	if strings.HasPrefix(data, "extra_toggle:") {
		b.handleExtraToggleCallback(callback)
		return
	}

	if callback.Message == nil {
		answer := tgbotapi.NewCallback(callback.ID, "Сообщение недоступно")
		b.api.Request(answer)
		return
	}

	chatID := callback.Message.Chat.ID

	if !b.isActiveInlineMessage(chatID, callback.Message.MessageID) {
		b.answerCallback(callback.ID, "Действие устарело. Открой меню заново.")
		return
	}

	if strings.HasPrefix(data, "edit_shift:") ||
		strings.HasPrefix(data, "edit_time:") ||
		strings.HasPrefix(data, "edit_break:") ||
		strings.HasPrefix(data, "edit_rate:") ||
		data == "cancel_edit" {
		b.handleEditCallback(callback)
		return
	}

	if strings.HasPrefix(data, "confirm_delete:") ||
		strings.HasPrefix(data, "delete_shift:") ||
		data == "cancel_delete" {
		b.handleDeleteCallback(callback)
		return
	}

	b.answerCallback(
		callback.ID,
		"Неизвестное действие. Открой меню заново.",
	)
}

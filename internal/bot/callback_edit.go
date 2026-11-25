package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleEditCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	if strings.HasPrefix(data, "edit_shift:") {
		shiftIDText := strings.TrimPrefix(data, "edit_shift:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			log.Printf("некорректный ID смены в callback %q", data)
			b.answerCallback(callback.ID, "❌ Некорректный номер смены.")
			return
		}

		if callback.Message == nil ||
			callback.From.ID != callback.Message.Chat.ID {
			b.answerCallback(callback.ID, "Нет доступа")
			return
		}

		shift, err := b.shiftService.GetShiftByID(
			context.Background(),
			shiftID,
			callback.From.ID,
		)

		if err != nil || shift == nil {
			b.answerCallback(callback.ID, "Смена не найдена")
			return
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"🕐 Время",
					fmt.Sprintf("edit_time:%d", shiftID),
				),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"☕ Перерыв",
					fmt.Sprintf("edit_break:%d", shiftID),
				),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"💰 Ставку",
					fmt.Sprintf("edit_rate:%d", shiftID),
				),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"◀️ Назад к смене",
					fmt.Sprintf("calendar:shift:%d", shiftID),
				),
			),
		)

		edit := tgbotapi.NewEditMessageText(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			fmt.Sprintf(
				"✏️ Смена #%d\n\nЧто изменить?",
				shiftID,
			),
		)

		edit.ReplyMarkup = &keyboard

		if _, err := b.api.Send(edit); err != nil {
			log.Println("ошибка открытия меню редактирования смены:", err)
			return
		}

		b.activeInlineMessages[callback.Message.Chat.ID] =
			callback.Message.MessageID

		answer := tgbotapi.NewCallback(
			callback.ID,
			"",
		)

		b.api.Request(answer)

		return
	}

	if strings.HasPrefix(data, "edit_time:") {
		shiftIDText := strings.TrimPrefix(data, "edit_time:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			log.Printf("некорректный ID смены в callback %q", data)
			b.answerCallback(callback.ID, "❌ Некорректный номер смены.")
			return
		}

		if callback.Message == nil ||
			callback.From.ID != callback.Message.Chat.ID {
			b.answerCallback(callback.ID, "Нет доступа")
			return
		}

		shift, err := b.shiftService.GetShiftByID(
			context.Background(),
			shiftID,
			callback.From.ID,
		)

		if err != nil || shift == nil {
			b.answerCallback(callback.ID, "Смена не найдена")
			return
		}

		b.clearActiveInlineKeyboard(callback.Message.Chat.ID)

		b.userStates[callback.Message.Chat.ID] = &UserState{
			Step:    "editing_start_time",
			ShiftID: shiftID,
		}

		msg := tgbotapi.NewMessage(
			callback.Message.Chat.ID,
			"🕐 Введи новое время начала смены в формате ЧЧ:ММ:",
		)

		msg.ReplyMarkup = mainKeyboard()

		b.api.Send(msg)

		answer := tgbotapi.NewCallback(callback.ID, "")
		b.api.Request(answer)

		return
	}

	if strings.HasPrefix(data, "edit_break:") {
		shiftIDText := strings.TrimPrefix(data, "edit_break:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			log.Printf("некорректный ID смены в callback %q", data)
			b.answerCallback(callback.ID, "❌ Некорректный номер смены.")
			return
		}

		b.clearActiveInlineKeyboard(callback.Message.Chat.ID)

		b.userStates[callback.Message.Chat.ID] = &UserState{
			Step:    "editing_break",
			ShiftID: shiftID,
		}

		msg := tgbotapi.NewMessage(
			callback.Message.Chat.ID,
			"☕ Введи новую продолжительность перерыва в минутах:\n\nНапример: 60",
		)

		msg.ReplyMarkup = mainKeyboard()

		b.api.Send(msg)

		answer := tgbotapi.NewCallback(callback.ID, "")
		b.api.Request(answer)

		return
	}

	if strings.HasPrefix(data, "edit_rate:") {
		shiftIDText := strings.TrimPrefix(data, "edit_rate:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			log.Printf("некорректный ID смены в callback %q", data)
			b.answerCallback(callback.ID, "❌ Некорректный номер смены.")
			return
		}

		b.clearActiveInlineKeyboard(callback.Message.Chat.ID)

		b.userStates[callback.Message.Chat.ID] = &UserState{
			Step:    "editing_rate",
			ShiftID: shiftID,
		}

		msg := tgbotapi.NewMessage(
			callback.Message.Chat.ID,
			"💰 Введи новую часовую ставку в рублях:\n\nНапример: 300",
		)

		msg.ReplyMarkup = mainKeyboard()

		b.api.Send(msg)

		answer := tgbotapi.NewCallback(callback.ID, "")
		b.api.Request(answer)

		return
	}

	if data == "cancel_edit" {
		b.clearActiveInlineKeyboard(callback.Message.Chat.ID)
		delete(b.userStates, callback.Message.Chat.ID)

		edit := tgbotapi.NewEditMessageText(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			"↩ Редактирование отменено.",
		)

		b.api.Send(edit)

		answer := tgbotapi.NewCallback(callback.ID, "")
		b.api.Request(answer)

		return
	}
}

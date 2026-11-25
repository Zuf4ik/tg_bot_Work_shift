package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleDeleteCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	if strings.HasPrefix(data, "confirm_delete:") {
		shiftIDText := strings.TrimPrefix(data, "confirm_delete:")

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
					"❌ Удалить",
					fmt.Sprintf("delete_shift:%d", shiftID),
				),
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
				"🗑 Ты точно хочешь удалить смену #%d?",
				shiftID,
			),
		)

		edit.ReplyMarkup = &keyboard

		b.api.Send(edit)

		answer := tgbotapi.NewCallback(
			callback.ID,
			"",
		)

		b.api.Request(answer)
		return
	}

	if strings.HasPrefix(data, "delete_shift:") {
		shiftIDText := strings.TrimPrefix(data, "delete_shift:")

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

		month := shift.StartTime.In(
			time.FixedZone("Уфа", 5*60*60),
		).Format("2006-01")

		err = b.shiftService.DeleteShift(
			context.Background(),
			shiftID,
			callback.From.ID,
		)

		if err != nil {
			log.Println("ошибка удаления смены:", err)

			answer := tgbotapi.NewCallback(
				callback.ID,
				"❌ Не удалось удалить смену",
			)

			b.api.Request(answer)
			return
		}

		b.clearActiveInlineKeyboard(callback.Message.Chat.ID)

		edit := tgbotapi.NewEditMessageText(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			fmt.Sprintf(
				"✅ Смена #%d удалена.",
				shiftID,
			),
		)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"🗓 Вернуться к календарю",
					"calendar:month:"+month,
				),
			),
		)

		edit.ReplyMarkup = &keyboard

		if _, err := b.api.Send(edit); err != nil {
			log.Println("ошибка обновления сообщения после удаления:", err)
			b.answerCallback(callback.ID, "Смена удалена, но не удалось обновить сообщение")
			return
		}

		// Регистрируем сообщение с кнопкой возврата как активное.
		b.activeInlineMessages[callback.Message.Chat.ID] =
			callback.Message.MessageID

		answer := tgbotapi.NewCallback(
			callback.ID,
			"Смена удалена",
		)

		b.api.Request(answer)
		return
	}

	if data == "cancel_delete" {
		b.clearActiveInlineKeyboard(callback.Message.Chat.ID)

		edit := tgbotapi.NewEditMessageText(
			callback.Message.Chat.ID,
			callback.Message.MessageID,
			"↩ Удаление отменено.",
		)

		if _, err := b.api.Send(edit); err != nil {
			log.Println("не удалось обновить сообщение об отмене удаления:", err)
		}

		answer := tgbotapi.NewCallback(callback.ID, "")
		if _, err := b.api.Request(answer); err != nil {
			log.Println("не удалось обработать callback отмены удаления:", err)
		}

		return
	}
}

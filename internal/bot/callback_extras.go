package bot

import (
	"context"
	"log"
	"strconv"
	"strings"

	"tg_bot/internal/repository"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleExtraToggleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	if strings.HasPrefix(data, "extra_toggle:") {
		if callback.Message == nil {
			b.answerCallback(callback.ID, "Сообщение недоступно")
			return
		}

		chatID := callback.Message.Chat.ID

		if !b.isActiveInlineMessage(chatID, callback.Message.MessageID) {
			b.answerCallback(callback.ID, "Меню устарело. Открой его заново.")
			return
		}

		if callback.From.ID != chatID {
			b.answerCallback(callback.ID, "Нет доступа")
			return
		}

		parts := strings.Split(data, ":")
		if len(parts) != 3 {
			b.answerCallback(callback.ID, "Некорректные данные")
			return
		}

		shiftID, err := strconv.Atoi(parts[1])
		if err != nil || shiftID <= 0 {
			b.answerCallback(callback.ID, "Некорректный ID смены")
			return
		}

		extraType := repository.ExtraType(parts[2])

		switch extraType {
		case repository.ExtraUnloading,
			repository.ExtraTrash,
			repository.ExtraDelivery:
		default:
			b.answerCallback(callback.ID, "Неизвестный тип работы")
			return
		}

		// Проверяем, что смена принадлежит пользователю.
		shift, err := b.shiftService.GetShiftByID(
			context.Background(),
			shiftID,
			callback.From.ID,
		)
		if err != nil || shift == nil {
			b.answerCallback(callback.ID, "Смена не найдена")
			return
		}

		extras, err := b.extraRepo.GetExtras(
			context.Background(),
			callback.From.ID,
			shiftID,
		)
		if err != nil {
			log.Println("Ошибка получения доплат:", err)
			b.answerCallback(callback.ID, "Ошибка базы данных")
			return
		}

		alreadySelected := false
		for _, extra := range extras {
			if extra == extraType {
				alreadySelected = true
				break
			}
		}

		if alreadySelected {
			err = b.extraRepo.DeleteExtra(
				context.Background(),
				callback.From.ID,
				shiftID,
				extraType,
			)
		} else {
			err = b.extraRepo.AddExtra(
				context.Background(),
				callback.From.ID,
				shiftID,
				extraType,
			)
		}

		if err != nil {
			log.Println("Ошибка изменения дополнительной работы:", err)
			b.answerCallback(callback.ID, "Не удалось сохранить изменение")
			return
		}

		b.answerCallback(callback.ID, "Сохранено")

		// Обновляем галочки в том же сообщении.
		b.handleExtraShift(callback, shiftID)
		return
	}
}

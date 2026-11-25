package bot

import (
	"context"
	"fmt"
	"log"

	"tg_bot/internal/repository"
	"tg_bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleExtraShift(
	callback *tgbotapi.CallbackQuery,
	shiftID int,
) {
	if callback.Message == nil {
		b.answerCallback(callback.ID, "Сообщение недоступно")
		return
	}

	chatID := callback.Message.Chat.ID

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
		log.Println("Ошибка получения дополнительных работ:", err)
		b.answerCallback(callback.ID, "Ошибка загрузки")
		return
	}

	baseSalary, err := service.CalculateSalary(*shift)
	if err != nil {
		log.Println("Ошибка расчёта зарплаты:", err)
		b.answerCallback(callback.ID, "Ошибка расчёта зарплаты")
		return
	}

	extraSalary, err := service.CalculateExtraSalary(*shift, extras)
	if err != nil {
		log.Println("Ошибка расчёта доплат:", err)
		b.answerCallback(callback.ID, "Ошибка расчёта доплат")
		return
	}

	totalSalary := baseSalary + extraSalary

	selected := make(map[repository.ExtraType]bool)
	for _, extra := range extras {
		selected[extra] = true
	}

	mark := func(extra repository.ExtraType) string {
		if selected[extra] {
			return "✅ "
		}
		return "⬜ "
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				mark(repository.ExtraUnloading)+"📦 Разгрузка",
				fmt.Sprintf("extra_toggle:%d:unloading", shiftID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				mark(repository.ExtraTrash)+"🗑 Мусор",
				fmt.Sprintf("extra_toggle:%d:trash", shiftID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				mark(repository.ExtraDelivery)+"🚚 Доставка",
				fmt.Sprintf("extra_toggle:%d:delivery", shiftID),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"◀️ Назад к смене",
				fmt.Sprintf("calendar:shift:%d", shiftID),
			),
		),
	)

	text := fmt.Sprintf(
		"📦 Дополнительные работы\n\n"+
			"📅 Смена: %s\n"+
			"⏰ %s–%s\n\n"+
			"💰 Основная зарплата: %.2f ₽\n"+
			"📦 Доплаты: %.2f ₽\n"+
			"💵 Итого за смену: %.2f ₽\n\n"+
			"Нажми на работу, чтобы добавить или убрать её:",
		shift.StartTime.Format("02.01.2006"),
		shift.StartTime.Format("15:04"),
		shift.EndTime.Format("15:04"),
		baseSalary,
		extraSalary,
		totalSalary,
	)

	edit := tgbotapi.NewEditMessageText(
		chatID,
		callback.Message.MessageID,
		text,
	)
	edit.ReplyMarkup = &keyboard

	if _, err := b.api.Send(edit); err != nil {
		log.Println("Ошибка открытия дополнительных работ:", err)
	}
}

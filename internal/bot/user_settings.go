package bot

import (
	"context"
	"log"
	"math"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleSetRate(message *tgbotapi.Message) {
	state, exists := b.userStates[message.Chat.ID]

	if !exists {
		state = &UserState{}
		b.userStates[message.Chat.ID] = state
	}

	state.Step = "waiting_hourly_rate"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"💰 Введи свою часовую ставку в рублях:\n\nНапример: 250",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

func (b *Bot) handleHourlyRate(message *tgbotapi.Message) {

	rate, err := strconv.ParseFloat(strings.TrimSpace(message.Text), 64)
	if err != nil || math.IsNaN(rate) || math.IsInf(rate, 0) || rate <= 0 || rate > 10000 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверная ставка. Введи сумму больше 0 и не более 10000 ₽/час.",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	if err := b.userRepo.SetHourlyRate(
		context.Background(),
		message.Chat.ID,
		rate,
	); err != nil {
		log.Println("ошибка сохранения ставки:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось сохранить ставку. Попробуй ещё раз.",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	log.Printf(
		"Пользователь %d указал ставку: %.2f ₽/час",
		message.Chat.ID,
		rate,
	)

	delete(b.userStates, message.Chat.ID)

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"✅ Часовая ставка сохранена: "+strconv.FormatFloat(rate, 'f', 2, 64)+" ₽/час",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

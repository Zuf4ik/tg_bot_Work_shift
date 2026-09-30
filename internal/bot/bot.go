package bot

import (
	"log"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserState struct {
	Step      string
	StartTime time.Time
}

type Bot struct {
	api        *tgbotapi.BotAPI
	userStates map[int64]*UserState
}

func New(api *tgbotapi.BotAPI) *Bot {
	return &Bot{
		api:        api,
		userStates: make(map[int64]*UserState),
	}
}

func (b *Bot) Start() {
	log.Println("Telegram bot handler started")

	config := tgbotapi.NewUpdate(0)
	config.Timeout = 60

	updates := b.api.GetUpdatesChan(config)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			b.handleStart(update.Message)
		}
		if update.Message.Text == "➕ Добавить смену" {
			b.handleAddShift(update.Message)
		}
		state, exists := b.userStates[update.Message.Chat.ID]

		if exists && state.Step == "waiting_start_time" {
		}
		if exists && state.Step == "waiting_end_time" {
			b.handleEndTime(update.Message)
		}
		if exists && state.Step == "waiting_hourly_rate" {
			b.handleHourlyRate(update.Message)
		}
	}
}
func mainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить смену"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("💰 Указать ставку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 История"),
			tgbotapi.NewKeyboardButton("📊 Статистика"),
		),
	)
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	text := "Привет! Я бот для учёта смен.\n\nВыбери действие:"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		text,
	)

	msg.ReplyMarkup = mainKeyboard()

	if _, err := b.api.Send(msg); err != nil {
		log.Println("Ошибка отправки сообщения:", err)
	}
}

func (b *Bot) handleAddShift(message *tgbotapi.Message) {
	b.userStates[message.Chat.ID] = &UserState{
		Step: "waiting_start_time",
	}
	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"Добавление смены.\n\nВведи время начала смены в формате ЧЧ:ММ:",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}

}

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
	rate, err := strconv.ParseFloat(message.Text, 64)
	if err != nil || rate <= 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверная ставка.\n\nВведи положительное число, например: 250",
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

func (b *Bot) handleStartTime(message *tgbotapi.Message) {
	startTime, err := parseTime(message.Text)
	state := b.userStates[message.Chat.ID]
	state.StartTime = startTime
	state.Step = "waiting_end_time"
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 10:00",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	log.Printf(
		"Пользователь %d ввёл время начала: %s",
		message.Chat.ID,
		startTime.Format("15:04"),
	)

	state.StartTime = startTime
	state.Step = "waiting_end_time"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"Время начала: "+startTime.Format("15:04")+
			"\n\nТеперь введи время окончания смены в формате ЧЧ:ММ:",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

func (b *Bot) handleEndTime(message *tgbotapi.Message) {
	endTime, err := parseTime(message.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 18:00",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	state := b.userStates[message.Chat.ID]

	log.Printf(
		"Пользователь %d ввёл время окончания: %s",
		message.Chat.ID,
		endTime.Format("15:04"),
	)

	state.Step = "waiting_hourly_rate"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"Время окончания: "+endTime.Format("15:04")+
			"\n\nТеперь введи часовую ставку в рублях.\nНапример: 250",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

func parseTime(value string) (time.Time, error) {
	return time.Parse("15:04", value)
}

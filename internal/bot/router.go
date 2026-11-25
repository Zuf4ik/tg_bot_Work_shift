package bot

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) Start() {
	log.Println("Telegram bot handler started")

	config := tgbotapi.NewUpdate(0)
	config.Timeout = 60

	updates := b.api.GetUpdatesChan(config)

	for update := range updates {
		if update.CallbackQuery != nil {
			b.handleCallback(update.CallbackQuery)
			continue
		}

		if update.Message == nil || update.Message.Chat == nil || update.Message.From == nil {
			continue
		}
		// Бот хранит состояние по ID чата; в группах это не ID пользователя.
		if update.Message.Chat.Type != "private" || update.Message.Chat.ID != update.Message.From.ID {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleStart(update.Message)
			continue
		}

		if update.Message.Text == "➕ Добавить смену" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleShiftCalendarForMode(update.Message, CalendarModeAdd)
			continue
		}

		if update.Message.Text == "📦 Дополнительные работы" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleShiftCalendarForMode(update.Message, CalendarModeExtras)
			continue
		}

		if update.Message.Text == "📊 Статистика" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleIncomeMenu(update.Message)
			continue
		}

		if update.Message.Text == "📅 Текущий месяц" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleCurrentMonthIncome(update.Message)
			continue
		}

		if update.Message.Text == "↩ Прошлый месяц" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handlePreviousMonthIncome(update.Message)
			continue
		}

		if update.Message.Text == "🗓 Свой период" {
			b.userStates[update.Message.Chat.ID] = &UserState{Step: "waiting_income_start"}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Введи начальную дату периода в формате ДД.ММ.ГГГГ. Например: 01.10.2026")
			msg.ReplyMarkup = incomePeriodKeyboard()
			b.api.Send(msg)
			continue
		}

		if update.Message.Text == "🗓 Календарь смен" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleShiftCalendar(update.Message)
			continue
		}

		if update.Message.Text == "💰 Указать ставку" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleSetRate(update.Message)
			continue
		}

		if update.Message.Text == "✏️ Редактировать смену" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleShiftCalendarForMode(update.Message, CalendarModeEdit)
			continue
		}

		if update.Message.Text == "🗑 Удалить смену" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleShiftCalendarForMode(update.Message, CalendarModeDelete)
			continue
		}
		if update.Message.Text == "🏠 Главное меню" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleStart(update.Message)
			continue
		}

		state, exists := b.userStates[update.Message.Chat.ID]

		if !exists {
			continue
		}

		if state.Step == "waiting_income_start" {
			b.handleIncomeStartDate(update.Message)
			continue
		}

		if state.Step == "waiting_income_end" {
			b.handleIncomeEndDate(update.Message)
			continue
		}

		if state.Step == "waiting_start_time" {
			b.handleStartTime(update.Message)
			continue
		}

		if state.Step == "editing_start_time" {
			b.handleEditingStartTime(update.Message)
			continue
		}

		if state.Step == "editing_end_time" {
			b.handleEditingEndTime(update.Message)
			continue
		}

		if state.Step == "editing_break" {
			b.handleEditingBreak(update.Message)
			continue
		}

		if state.Step == "editing_rate" {
			b.handleEditingRate(update.Message)
			continue
		}

		if state.Step == "waiting_end_time" {
			b.handleEndTime(update.Message)
			continue
		}

		if state.Step == "waiting_hourly_rate" {
			b.handleHourlyRate(update.Message)
			continue
		}

		if state.Step == "waiting_break" {
			b.handleBreakTime(update.Message)
			continue
		}
	}
}

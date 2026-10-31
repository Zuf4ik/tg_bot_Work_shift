package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"tg_bot/internal/repository"
	"tg_bot/model"
	"tg_bot/service"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserState struct {
	ShiftID     int
	Step        string
	StartTime   time.Time
	EndTime     time.Time
	ReportStart time.Time
}

type Bot struct {
	api                  *tgbotapi.BotAPI
	userRepo             *repository.UserRepository
	userStates           map[int64]*UserState
	shiftService         *service.ShiftService
	salaryService        *service.SalaryService
	activeInlineMessages map[int64]int
}

func New(
	api *tgbotapi.BotAPI,
	userRepo *repository.UserRepository,
	shiftService *service.ShiftService,
	salaryService *service.SalaryService,
) *Bot {
	return &Bot{
		api:                  api,
		userRepo:             userRepo,
		shiftService:         shiftService,
		salaryService:        salaryService,
		userStates:           make(map[int64]*UserState),
		activeInlineMessages: make(map[int64]int),
	}
}

// Убирает inline-клавиатуру и забывает активное сообщение чата.
func (b *Bot) clearActiveInlineKeyboard(chatID int64) {
	messageID, exists := b.activeInlineMessages[chatID]
	if !exists {
		return
	}

	markup := tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: [][]tgbotapi.InlineKeyboardButton{},
	}

	edit := tgbotapi.NewEditMessageReplyMarkup(
		chatID,
		messageID,
		markup,
	)

	if _, err := b.api.Request(edit); err != nil {
		log.Println("не удалось убрать inline-клавиатуру:", err)
	}

	delete(b.activeInlineMessages, chatID)
}

// Завершает текущий сценарий перед переходом в меню.
func (b *Bot) resetUserFlow(chatID int64) {
	b.clearActiveInlineKeyboard(chatID)
	delete(b.userStates, chatID)
}

// Отвечает на нажатие inline-кнопки и записывает ошибку, если ответ не отправился.
func (b *Bot) answerCallback(callbackID, text string) {
	answer := tgbotapi.NewCallback(callbackID, text)

	if _, err := b.api.Request(answer); err != nil {
		log.Println("не удалось обработать callback:", err)
	}
}

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

		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleStart(update.Message)
			continue
		}

		if update.Message.Text == "➕ Добавить смену" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleAddShift(update.Message)
			continue
		}

		if update.Message.Text == "💵 Доход за период" {
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

		if update.Message.Text == "📋 История" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleHistory(update.Message)
			continue
		}

		if update.Message.Text == "💰 Указать ставку" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleSetRate(update.Message)
			continue
		}

		if update.Message.Text == "✏️ Редактировать смену" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleEditShift(update.Message)
			continue
		}

		if update.Message.Text == "🗑 Удалить смену" {
			b.resetUserFlow(update.Message.Chat.ID)
			b.handleDeleteShift(update.Message)
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

		if state.Step == "waiting_shift_date" {
			b.handleShiftDate(update.Message)
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

func mainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("➕ Добавить смену"),
			tgbotapi.NewKeyboardButton("💰 Указать ставку"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📋 История"),
			tgbotapi.NewKeyboardButton("💵 Доход за период"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("✏️ Редактировать смену"),
			tgbotapi.NewKeyboardButton("🗑 Удалить смену"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🏠 Главное меню"),
		),
	)

	keyboard.ResizeKeyboard = true
	return keyboard
}

func incomePeriodKeyboard() tgbotapi.ReplyKeyboardMarkup {
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("📅 Текущий месяц")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("↩ Прошлый месяц")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🗓 Свой период")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("🏠 Главное меню")),
	)
	keyboard.ResizeKeyboard = true
	return keyboard
}

func (b *Bot) handleIncomeMenu(message *tgbotapi.Message) {
	delete(b.userStates, message.Chat.ID)
	msg := tgbotapi.NewMessage(message.Chat.ID, "💵 За какой период показать доход?")
	msg.ReplyMarkup = incomePeriodKeyboard()
	if _, err := b.api.Send(msg); err != nil {
		log.Println("не удалось показать выбор периода дохода:", err)
	}
}

func (b *Bot) handleCurrentMonthIncome(message *tgbotapi.Message) {
	now := time.Now().In(time.Local)
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	b.sendIncomeReport(message.Chat.ID, start, end)
}

func (b *Bot) handlePreviousMonthIncome(message *tgbotapi.Message) {
	now := time.Now().In(time.Local)
	thisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	start := thisMonth.AddDate(0, -1, 0)
	end := thisMonth.Add(-time.Nanosecond)
	b.sendIncomeReport(message.Chat.ID, start, end)
}

func (b *Bot) handleIncomeStartDate(message *tgbotapi.Message) {
	date, err := time.ParseInLocation("02.01.2006", strings.TrimSpace(message.Text), time.Local)
	if err != nil {
		b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Не понял дату. Введи её в формате ДД.ММ.ГГГГ, например 01.10.2026."))
		return
	}
	if date.Year() < 2020 || date.Year() > 2036 {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Год должен быть в диапазоне от 2020 до 2036.",
		))
		return
	}
	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}
	state.ReportStart = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
	state.Step = "waiting_income_end"
	b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "Теперь введи конечную дату периода в формате ДД.ММ.ГГГГ. Например: 31.10.2026"))
}

func (b *Bot) handleIncomeEndDate(message *tgbotapi.Message) {
	date, err := time.ParseInLocation("02.01.2006", strings.TrimSpace(message.Text), time.Local)
	if err != nil {
		b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Не понял дату. Введи её в формате ДД.ММ.ГГГГ, например 31.10.2026."))
		return
	}
	if date.Year() < 2020 || date.Year() > 2036 {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Год должен быть в диапазоне от 2020 до 2036.",
		))
		return
	}
	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}
	end := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.Local)
	start := state.ReportStart
	if end.Before(start) {
		b.api.Send(tgbotapi.NewMessage(message.Chat.ID, "❌ Конечная дата раньше начальной. Введи конечную дату ещё раз."))
		return
	}
	delete(b.userStates, message.Chat.ID)
	b.sendIncomeReport(message.Chat.ID, start, end)
}

func (b *Bot) sendIncomeReport(chatID int64, start, end time.Time) {

	shifts, err := b.shiftService.GetShifts(context.Background(), chatID)
	if err != nil {
		log.Println("ошибка получения смен для отчёта:", err)
		b.api.Send(tgbotapi.NewMessage(chatID, "❌ Не удалось сформировать отчёт о доходах."))
		return
	}

	now := time.Now()
	var actualSalary, forecastSalary, actualHours, forecastHours float64
	var actualCount, forecastCount int

	for _, shift := range shifts {
		// В отчёт попадают смены, дата начала которых входит в выбранный диапазон.

		if shift.StartTime.Before(start) || shift.StartTime.After(end) {
			continue
		}
		salary, salaryErr := service.CalculateSalary(shift)
		hours, hoursErr := service.CalculateHours(shift)
		if salaryErr != nil || hoursErr != nil {
			log.Printf("пропущена смена #%d при расчёте отчёта: salary=%v hours=%v", shift.ID, salaryErr, hoursErr)
			continue
		}
		if !shift.EndTime.After(now) {
			actualSalary += salary
			actualHours += hours
			actualCount++
		} else {
			forecastSalary += salary
			forecastHours += hours
			forecastCount++
		}
	}

	text := fmt.Sprintf(
		"💵 Доход за период\n📅 %s — %s\n\n"+
			"✅ Заработано (завершённые смены): %.2f ₽\n"+
			"   Смен: %d · Часов: %.2f\n\n"+
			"🔮 Ожидается (незавершённые/плановые смены): %.2f ₽\n"+
			"   Смен: %d · Часов: %.2f\n\n"+
			"📊 Итого с учётом прогноза: %.2f ₽",
		start.Format("02.01.2006"), end.Format("02.01.2006"),
		actualSalary, actualCount, actualHours,
		forecastSalary, forecastCount, forecastHours,
		actualSalary+forecastSalary,
	)

	// Премию показываем только для полного календарного месяца.
	monthStart := time.Date(
		start.Year(), start.Month(), 1,
		0, 0, 0, 0, start.Location(),
	)
	nextMonth := monthStart.AddDate(0, 1, 0)

	if start.Equal(monthStart) &&
		!end.Before(nextMonth.Add(-time.Second)) &&
		end.Before(nextMonth) {

		var completedShifts []model.Shift
		var allMonthShifts []model.Shift

		for _, shift := range shifts {
			if shift.StartTime.Before(monthStart) ||
				!shift.StartTime.Before(nextMonth) {
				continue
			}

			allMonthShifts = append(allMonthShifts, shift)

			if !shift.EndTime.After(now) {
				completedShifts = append(completedShifts, shift)
			}
		}

		actualBonus, actualErr := b.salaryService.CalculateBonusFromShifts(
			context.Background(),
			monthStart.Year(),
			monthStart.Month(),
			completedShifts,
		)

		forecastBonus, forecastErr := b.salaryService.CalculateBonusFromShifts(
			context.Background(),
			monthStart.Year(),
			monthStart.Month(),
			allMonthShifts,
		)

		if actualErr != nil || forecastErr != nil {
			log.Printf(
				"ошибка расчёта месячной премии: actual=%v forecast=%v",
				actualErr, forecastErr,
			)
			text += "\n\n⚠️ Не удалось рассчитать месячную премию."
		} else {
			text += fmt.Sprintf(
				"\n\n📊 Месячная статистика\n"+
					"⏱ Отработано: %.2f ч\n"+
					"📅 Норма: %.2f ч\n"+
					"⏳ До нормы: %.2f ч\n"+
					"💎 Премия за завершённые смены: %.2f ₽\n"+
					"🔮 Премия с учётом плана: %.2f ₽",
				actualBonus.WorkedHours,
				actualBonus.NormHours,
				actualBonus.Remaining,
				actualBonus.Bonus,
				forecastBonus.Bonus,
			)
		}
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = mainKeyboard()
	if _, err := b.api.Send(msg); err != nil {
		log.Println("не удалось отправить отчёт о доходах:", err)
	}
}

func (b *Bot) handleStart(message *tgbotapi.Message) {
	text := "Привет! Я бот для учёта смен.\n\nПеред добавлением первой смены сначала укажи свою часовую ставку через кнопку «💰 Указать ставку». Она нужна для расчёта зарплаты. После этого нажми «➕ Добавить смену».\n\nВыбери действие:"

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
		Step: "waiting_shift_date",
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"Добавление смены.\n\nВведи дату смены в формате ДД.ММ.ГГГГ.\nНапример: 14.10.2026",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

// handleShiftDate сохраняет дату смены и переходит к вводу времени начала.
func (b *Bot) handleShiftDate(message *tgbotapi.Message) {
	shiftDate, err := time.ParseInLocation("02.01.2006", strings.TrimSpace(message.Text), time.Local)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверный формат даты.\n\nВведи дату в формате ДД.ММ.ГГГГ, например: 14.10.2026",
		)
		if _, sendErr := b.api.Send(msg); sendErr != nil {
			log.Println("ошибка отправки сообщения:", sendErr)
		}
		return
	}

	if shiftDate.Year() < 2020 || shiftDate.Year() > 2036 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Год смены должен быть в диапазоне от 2020 до 2036.",
		)
		if _, sendErr := b.api.Send(msg); sendErr != nil {
			log.Println("ошибка отправки сообщения:", sendErr)
		}
		return
	}

	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}

	state.StartTime = time.Date(
		shiftDate.Year(),
		shiftDate.Month(),
		shiftDate.Day(),
		0, 0, 0, 0,
		time.Local,
	)
	state.Step = "waiting_start_time"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"📅 Дата смены: "+shiftDate.Format("02.01.2006")+
			"\n\nТеперь введи время начала смены в формате ЧЧ:ММ, например: 10:00",
	)
	if _, sendErr := b.api.Send(msg); sendErr != nil {
		log.Println("ошибка отправки сообщения:", sendErr)
	}
}

func (b *Bot) handleHistory(message *tgbotapi.Message) {
	ctx := context.Background()

	shifts, err := b.shiftService.GetShifts(
		ctx,
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения истории:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось получить историю смен.",
		)

		b.api.Send(msg)
		return
	}

	if len(shifts) == 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"📋 История смен пока пуста.",
		)

		b.api.Send(msg)
		return
	}

	text := "📋 История смен:\n\n"

	for i, shift := range shifts {
		hours, err := service.CalculateHours(shift)
		if err != nil {
			continue
		}

		salary, err := service.CalculateSalary(shift)
		if err != nil {
			continue
		}

		text += fmt.Sprintf(
			"%d. 📅 %s\n"+
				"🕐 %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ %.2f ч.\n"+
				"💰 %.2f ₽\n\n",
			i+1,
			shift.StartTime.Format("02.01.2006"),
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			int(shift.Break.Minutes()),
			hours,
			salary,
		)
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		text,
	)

	b.api.Send(msg)
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

	rate, err := strconv.ParseFloat(strings.TrimSpace(message.Text), 64)
	if err != nil || rate <= 0 || rate > 10000 {
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

func (b *Bot) handleStartTime(message *tgbotapi.Message) {
	startTime, err := parseTime(message.Text)

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

	state := b.userStates[message.Chat.ID]
	if state == nil {
		return
	}

	// Используем дату, которую пользователь ввёл перед временем начала.
	startTime = time.Date(
		state.StartTime.Year(),
		state.StartTime.Month(),
		state.StartTime.Day(),
		startTime.Hour(),
		startTime.Minute(),
		0,
		0,
		time.Local,
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
	if state == nil {
		return
	}

	endTime = time.Date(
		state.StartTime.Year(),
		state.StartTime.Month(),
		state.StartTime.Day(),
		endTime.Hour(),
		endTime.Minute(),
		0,
		0,
		state.StartTime.Location(),
	)

	if endTime.Before(state.StartTime) {
		endTime = endTime.Add(24 * time.Hour)
	}

	state.EndTime = endTime

	log.Printf(
		"Пользователь %d ввёл время окончания: %s",
		message.Chat.ID,
		endTime.Format("15:04"),
	)

	state.Step = "waiting_break"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"Время окончания: "+endTime.Format("15:04")+
			"\n\nТеперь введи продолжительность перерыва в минутах.\nНапример: 60",
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}
}

func (b *Bot) handleBreakTime(message *tgbotapi.Message) {
	breakMinutes, err := strconv.Atoi(message.Text)
	if err != nil || breakMinutes < 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверное значение.\n\nВведи количество минут, например: 60",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	state := b.userStates[message.Chat.ID]

	rate, err := b.userRepo.GetHourlyRate(
		context.Background(),
		message.Chat.ID,
	)

	if err != nil || rate <= 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Часовая ставка не установлена.\n\nСначала нажми «💰 Указать ставку».",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift := model.Shift{
		TelegramID: message.Chat.ID,
		StartTime:  state.StartTime,
		EndTime:    state.EndTime,
		Break:      time.Duration(breakMinutes) * time.Minute,
		HourlyRate: rate,
	}

	hours, err := service.CalculateHours(shift)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось рассчитать продолжительность смены: "+err.Error(),
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	salary, err := service.CalculateSalary(shift)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось рассчитать зарплату: "+err.Error(),
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	_, err = b.shiftService.CreateShift(
		context.Background(),
		shift,
	)

	if err != nil {
		log.Println("ошибка сохранения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось сохранить смену: "+err.Error(),
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка отправки сообщения:", err)
		}

		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена рассчитана!\n\n"+
				"📅 Дата: %s\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Ставка: %.2f ₽/час\n"+
				"💵 Зарплата: %.2f ₽",
			shift.StartTime.Format("02.01.2006"),
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			breakMinutes,
			hours,
			rate,
			salary,
		),
	)

	if _, err := b.api.Send(msg); err != nil {
		log.Println("ошибка отправки сообщения:", err)
	}

	delete(b.userStates, message.Chat.ID)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse("15:04", value)
}

func (b *Bot) handleDeleteShift(message *tgbotapi.Message) {
	shifts, err := b.shiftService.GetShifts(
		context.Background(),
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смен:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось получить список смен.",
		)

		b.api.Send(msg)
		return
	}

	if len(shifts) == 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"📋 У тебя пока нет смен для удаления.",
		)

		b.api.Send(msg)
		return
	}

	keyboard := make([][]tgbotapi.InlineKeyboardButton, 0)

	for _, shift := range shifts {
		buttonText := fmt.Sprintf(
			"#%d | %s %s–%s",
			shift.ID,
			shift.StartTime.Format("02.01"),
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
		)

		button := tgbotapi.NewInlineKeyboardButtonData(
			buttonText,
			fmt.Sprintf("confirm_delete:%d", shift.ID),
		)

		keyboard = append(
			keyboard,
			tgbotapi.NewInlineKeyboardRow(button),
		)
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"🗑 Выбери смену, которую хочешь удалить:",
	)

	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	sentMessage, err := b.api.Send(msg)
	if err != nil {
		log.Println("не удалось отправить список смен для удаления:", err)
		return
	}

	b.activeInlineMessages[message.Chat.ID] = sentMessage.MessageID
}

func (b *Bot) handleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data

	if callback.Message == nil {
		answer := tgbotapi.NewCallback(callback.ID, "Сообщение недоступно")
		b.api.Request(answer)
		return
	}

	chatID := callback.Message.Chat.ID
	activeMessageID, exists := b.activeInlineMessages[chatID]

	if !exists || activeMessageID != callback.Message.MessageID {
		answer := tgbotapi.NewCallback(
			callback.ID,
			"Действие устарело. Открой меню заново.",
		)
		b.api.Request(answer)
		return
	}

	if strings.HasPrefix(data, "edit_shift:") {
		shiftIDText := strings.TrimPrefix(data, "edit_shift:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			log.Printf("некорректный ID смены в callback %q", data)
			b.answerCallback(callback.ID, "❌ Некорректный номер смены.")
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
					"↩ Отмена",
					"cancel_edit",
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

		b.api.Send(edit)

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

	if strings.HasPrefix(data, "confirm_delete:") {
		shiftIDText := strings.TrimPrefix(data, "confirm_delete:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			log.Printf("некорректный ID смены в callback %q", data)
			b.answerCallback(callback.ID, "❌ Некорректный номер смены.")
			return
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					"❌ Удалить",
					fmt.Sprintf("delete_shift:%d", shiftID),
				),
				tgbotapi.NewInlineKeyboardButtonData(
					"↩ Отмена",
					"cancel_delete",
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

		err = b.shiftService.DeleteShift(
			context.Background(),
			shiftID,
			callback.Message.Chat.ID,
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

		b.api.Send(edit)

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

	b.answerCallback(
		callback.ID,
		"Неизвестное действие. Открой меню заново.",
	)
}

func (b *Bot) handleEditShift(message *tgbotapi.Message) {
	shifts, err := b.shiftService.GetShifts(
		context.Background(),
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смен:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось получить список смен.",
		)

		b.api.Send(msg)
		return
	}

	if len(shifts) == 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"📋 У тебя пока нет смен для редактирования.",
		)

		b.api.Send(msg)
		return
	}

	keyboard := make([][]tgbotapi.InlineKeyboardButton, 0)

	for _, shift := range shifts {
		buttonText := fmt.Sprintf(
			"#%d | %s %s–%s",
			shift.ID,
			shift.StartTime.Format("02.01"),
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
		)

		button := tgbotapi.NewInlineKeyboardButtonData(
			buttonText,
			fmt.Sprintf("edit_shift:%d", shift.ID),
		)

		keyboard = append(
			keyboard,
			tgbotapi.NewInlineKeyboardRow(button),
		)
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"✏️ Выбери смену, которую хочешь изменить:",
	)

	msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
		InlineKeyboard: keyboard,
	}

	sentMessage, err := b.api.Send(msg)
	if err != nil {
		log.Println("не удалось отправить список смен для редактирования:", err)
		return
	}

	b.activeInlineMessages[message.Chat.ID] = sentMessage.MessageID
}

func (b *Bot) handleEditingStartTime(message *tgbotapi.Message) {
	startTime, err := parseTime(message.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 10:00",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

	shifts, err := b.shiftService.GetShifts(
		context.Background(),
		message.Chat.ID,
	)
	if err != nil {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"Ошибка получения смены.",
		))
		return
	}

	var oldShift *model.Shift

	for i := range shifts {
		if shifts[i].ID == state.ShiftID {
			oldShift = &shifts[i]
			break
		}
	}

	if oldShift == nil {
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"Смена не найдена.",
		))
		delete(b.userStates, message.Chat.ID)
		return
	}

	state.StartTime = time.Date(
		oldShift.StartTime.Year(),
		oldShift.StartTime.Month(),
		oldShift.StartTime.Day(),
		startTime.Hour(),
		startTime.Minute(),
		0,
		0,
		oldShift.StartTime.Location(),
	)

	state.Step = "editing_end_time"

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		"🕐 Теперь введи новое время окончания смены в формате ЧЧ:ММ:",
	)

	msg.ReplyMarkup = mainKeyboard()

	b.api.Send(msg)
}

func (b *Bot) handleEditingEndTime(message *tgbotapi.Message) {
	endTime, err := parseTime(message.Text)
	if err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"Неверный формат времени.\n\nВведи время в формате ЧЧ:ММ, например: 18:00",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

	endTime = time.Date(
		state.StartTime.Year(),
		state.StartTime.Month(),
		state.StartTime.Day(),
		endTime.Hour(),
		endTime.Minute(),
		0,
		0,
		state.StartTime.Location(),
	)
	if endTime.Before(state.StartTime) {
		endTime = endTime.Add(24 * time.Hour)
	}

	state.EndTime = endTime

	shift := model.Shift{
		ID:         state.ShiftID,
		TelegramID: message.Chat.ID,
		StartTime:  state.StartTime,
		EndTime:    state.EndTime,
	}

	oldShift, err := b.shiftService.GetShiftByID(
		context.Background(),
		state.ShiftID,
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось найти смену.",
		)
		msg.ReplyMarkup = mainKeyboard()
		b.api.Send(msg)

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift.Break = oldShift.Break
	shift.HourlyRate = oldShift.HourlyRate

	// Проверяем новую смену перед сохранением.
	if _, err := service.CalculateHours(shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Некорректное время смены: "+err.Error(),
		)
		b.api.Send(msg)

		return
	}

	if _, err := service.CalculateSalary(shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось пересчитать зарплату: "+err.Error(),
		)
		b.api.Send(msg)

		return
	}

	err = b.shiftService.UpdateShift(
		context.Background(),
		state.ShiftID,
		message.From.ID,
		shift,
	)
	if err != nil {
		log.Println("ошибка обновления смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось обновить смену: "+err.Error(),
		)
		b.api.Send(msg)

		return
	}

	salary, err := service.CalculateSalary(shift)
	if err != nil {
		log.Println("ошибка расчёта зарплаты:", err)
		return
	}

	hours, err := service.CalculateHours(shift)
	if err != nil {
		log.Println("ошибка расчёта часов:", err)
		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена #%d обновлена!\n\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Зарплата: %.2f ₽",
			state.ShiftID,
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			int(shift.Break.Minutes()),
			hours,
			salary,
		),
	)

	msg.ReplyMarkup = mainKeyboard()

	b.api.Send(msg)

	delete(b.userStates, message.Chat.ID)
}

func (b *Bot) handleEditingBreak(message *tgbotapi.Message) {
	breakMinutes, err := strconv.Atoi(message.Text)
	if err != nil || breakMinutes < 0 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверное значение.\n\nВведи количество минут, например: 60",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

	shift, err := b.shiftService.GetShiftByID(
		context.Background(),
		state.ShiftID,
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось найти смену.",
		)
		b.api.Send(msg)

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift.Break = time.Duration(breakMinutes) * time.Minute

	if _, err := service.CalculateHours(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Нельзя установить такой перерыв: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	if _, err := service.CalculateSalary(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось пересчитать зарплату: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	err = b.shiftService.UpdateShift(
		context.Background(),
		state.ShiftID,
		message.From.ID,
		*shift,
	)
	if err != nil {
		log.Println("ошибка обновления перерыва:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось обновить перерыв.",
		)
		b.api.Send(msg)
		return
	}

	hours, err := service.CalculateHours(*shift)
	if err != nil {
		log.Println("ошибка расчёта часов:", err)
		return
	}

	salary, err := service.CalculateSalary(*shift)
	if err != nil {
		log.Println("ошибка расчёта зарплаты:", err)
		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена #%d обновлена!\n\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Зарплата: %.2f ₽",
			state.ShiftID,
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			breakMinutes,
			hours,
			salary,
		),
	)

	msg.ReplyMarkup = mainKeyboard()

	b.api.Send(msg)

	delete(b.userStates, message.Chat.ID)
}

func (b *Bot) handleEditingRate(message *tgbotapi.Message) {
	rate, err := strconv.ParseFloat(strings.TrimSpace(message.Text), 64)
	if err != nil || rate <= 0 || rate > 10000 {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Неверная ставка.\n\nВведи положительное число, например: 300",
		)
		b.api.Send(msg)
		return
	}

	state := b.userStates[message.Chat.ID]

	shift, err := b.shiftService.GetShiftByID(
		context.Background(),
		state.ShiftID,
		message.Chat.ID,
	)

	if err != nil {
		log.Println("ошибка получения смены:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось найти смену.",
		)
		b.api.Send(msg)

		delete(b.userStates, message.Chat.ID)
		return
	}

	shift.HourlyRate = rate

	if _, err := service.CalculateHours(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось рассчитать продолжительность смены: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	if _, err := service.CalculateSalary(*shift); err != nil {
		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось пересчитать зарплату: "+err.Error(),
		)
		b.api.Send(msg)
		return
	}

	err = b.shiftService.UpdateShift(
		context.Background(),
		state.ShiftID,
		message.From.ID,
		*shift,
	)
	if err != nil {
		log.Println("ошибка обновления ставки:", err)

		msg := tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось обновить ставку.",
		)
		b.api.Send(msg)
		return
	}

	hours, err := service.CalculateHours(*shift)
	if err != nil {
		log.Println("ошибка расчёта часов:", err)
		return
	}

	salary, err := service.CalculateSalary(*shift)
	if err != nil {
		log.Println("ошибка расчёта зарплаты:", err)
		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		fmt.Sprintf(
			"✅ Смена #%d обновлена!\n\n"+
				"🕐 Время: %s — %s\n"+
				"☕ Перерыв: %d мин.\n"+
				"⏱ Часы: %.2f\n"+
				"💰 Новая ставка: %.2f ₽/час\n"+
				"💵 Зарплата: %.2f ₽",
			state.ShiftID,
			shift.StartTime.Format("15:04"),
			shift.EndTime.Format("15:04"),
			int(shift.Break.Minutes()),
			hours,
			rate,
			salary,
		),
	)

	msg.ReplyMarkup = mainKeyboard()

	b.api.Send(msg)

	delete(b.userStates, message.Chat.ID)
}

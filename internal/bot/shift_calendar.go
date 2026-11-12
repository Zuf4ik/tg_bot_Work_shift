package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"tg_bot/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func calendarMonthStart(date time.Time) time.Time {
	return time.Date(
		date.Year(),
		date.Month(),
		1,
		0, 0, 0, 0,
		date.Location(),
	)
}

func buildShiftCalendar(
	date time.Time,
	busyDays map[int]bool,
) tgbotapi.InlineKeyboardMarkup {
	firstDay := calendarMonthStart(date)

	// Определяем количество дней в месяце.
	daysInMonth := time.Date(
		firstDay.Year(),
		firstDay.Month()+1,
		0, 0, 0, 0, 0,
		firstDay.Location(),
	).Day()

	// В Go неделя начинается с воскресенья.
	// Переводим её к привычному формату: понедельник = 0.
	offset := (int(firstDay.Weekday()) + 6) % 7

	rows := [][]tgbotapi.InlineKeyboardButton{}

	// Названия дней недели.
	weekdays := []string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}

	header := []tgbotapi.InlineKeyboardButton{}

	for _, day := range weekdays {
		header = append(
			header,
			tgbotapi.NewInlineKeyboardButtonData(day, "calendar:noop"),
		)
	}

	rows = append(rows, header)

	// Формируем строки календаря.
	week := []tgbotapi.InlineKeyboardButton{}

	for i := 0; i < offset; i++ {
		week = append(
			week,
			tgbotapi.NewInlineKeyboardButtonData(" ", "calendar:noop"),
		)
	}

	for day := 1; day <= daysInMonth; day++ {
		currentDate := time.Date(
			firstDay.Year(),
			firstDay.Month(),
			day, 0, 0, 0, 0,
			firstDay.Location(),
		)

		callback := fmt.Sprintf(
			"calendar:day:%s",
			currentDate.Format("2006-01-02"),
		)

		label := fmt.Sprintf("%d", day)

		if busyDays[day] {
			label = "🔹" + label
		}

		button := tgbotapi.NewInlineKeyboardButtonData(
			label,
			callback,
		)

		week = append(week, button)

		if len(week) == 7 {
			rows = append(rows, week)
			week = []tgbotapi.InlineKeyboardButton{}
		}
	}

	if len(week) > 0 {
		rows = append(rows, week)
	}

	// Определяем предыдущий и следующий месяцы.
	prevMonth := firstDay.AddDate(0, -1, 0)
	nextMonth := firstDay.AddDate(0, 1, 0)

	// Создаём кнопки переключения месяцев.
	prevButton := tgbotapi.NewInlineKeyboardButtonData(
		"◀️",
		"calendar:month:"+prevMonth.Format("2006-01"),
	)

	nextButton := tgbotapi.NewInlineKeyboardButtonData(
		"▶️",
		"calendar:month:"+nextMonth.Format("2006-01"),
	)

	// Добавляем строку навигации под календарём.
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		prevButton,
		nextButton,
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func (b *Bot) handleShiftCalendarCallback(callback *tgbotapi.CallbackQuery) {
	if callback.Message == nil {
		b.answerCallback(callback.ID, "Сообщение недоступно")
		return
	}

	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	// Проверяем, что пользователь нажал кнопку
	// действующего календаря.
	activeID, exists := b.activeInlineMessages[chatID]
	if !exists || activeID != messageID {
		b.answerCallback(callback.ID, "Календарь устарел")
		return
	}

	data := callback.Data

	if data == "calendar:noop" {
		b.answerCallback(callback.ID, "")
		return
	}

	// Обработку выбранной даты добавим следующим шагом.

	if strings.HasPrefix(data, "calendar:day:") {
		loc, err := time.LoadLocation("Asia/Yekaterinburg")
		if err != nil {
			b.answerCallback(callback.ID, "Ошибка часового пояса")
			return
		}

		dateText := strings.TrimPrefix(data, "calendar:day:")

		date, err := time.ParseInLocation(
			"2006-01-02",
			dateText,
			loc,
		)
		if err != nil || date.Year() < 2020 || date.Year() > 2036 {
			b.answerCallback(callback.ID, "Некорректная дата")
			return
		}

		// Получаем информацию о сменах за выбранный день.
		text, err := b.buildCalendarDayText(chatID, date)
		if err != nil {
			log.Println("ошибка получения смен за день:", err)
			b.answerCallback(callback.ID, "Ошибка загрузки смен")
			return
		}

		// Кнопка возврата к выбранному месяцу.
		backButton := tgbotapi.NewInlineKeyboardButtonData(
			"◀️ Назад к календарю",
			"calendar:month:"+date.Format("2006-01"),
		)

		addButton := tgbotapi.NewInlineKeyboardButtonData(
			"➕ Добавить смену",
			"calendar:add:"+date.Format("2006-01-02"),
		)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(addButton),
			tgbotapi.NewInlineKeyboardRow(backButton),
		)

		// Обновляем существующее сообщение.
		edit := tgbotapi.NewEditMessageText(
			chatID,
			messageID,
			text,
		)

		edit.ReplyMarkup = &keyboard

		if _, err := b.api.Send(edit); err != nil {
			log.Println("ошибка обновления календаря:", err)
			b.answerCallback(callback.ID, "Не удалось показать смены")
			return
		}

		b.answerCallback(callback.ID, "")
		return
	}

	if strings.HasPrefix(data, "calendar:add:") {
		loc, err := time.LoadLocation("Asia/Yekaterinburg")
		if err != nil {
			b.answerCallback(callback.ID, "Ошибка часового пояса")
			return
		}

		dateText := strings.TrimPrefix(data, "calendar:add:")

		date, err := time.ParseInLocation(
			"2006-01-02",
			dateText,
			loc,
		)
		if err != nil || date.Year() < 2020 || date.Year() > 2036 {
			b.answerCallback(callback.ID, "Некорректная дата")
			return
		}

		// Завершаем предыдущий сценарий.
		b.resetUserFlow(chatID)

		// Начинаем добавление смены с выбранной датой.
		b.userStates[chatID] = &UserState{
			Step:      "waiting_start_time",
			StartTime: startOfShiftDay(date),
		}

		msg := tgbotapi.NewMessage(
			chatID,
			"➕ Добавление смены\n\n"+
				"📅 Дата: "+date.Format("12.11.2026")+
				"\n\nВведи время начала смены в формате ЧЧ:ММ.",
		)

		if _, err := b.api.Send(msg); err != nil {
			log.Println("ошибка начала добавления смены:", err)
			delete(b.userStates, chatID)
			b.answerCallback(callback.ID, "Ошибка отправки сообщения")
			return
		}

		b.answerCallback(callback.ID, "")
		return
	}

	if !strings.HasPrefix(data, "calendar:month:") {
		b.answerCallback(callback.ID, "Неизвестная команда")
		return
	}

	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		b.answerCallback(callback.ID, "Ошибка часового пояса")
		return
	}

	monthText := strings.TrimPrefix(data, "calendar:month:")

	month, err := time.ParseInLocation(
		"2006-01",
		monthText,
		loc,
	)
	if err != nil || month.Year() < 2020 || month.Year() > 2036 {
		b.answerCallback(callback.ID, "Некорректный месяц")
		return
	}

	busyDays, err := b.getCalendarBusyDays(chatID, month)
	if err != nil {
		log.Println("ошибка получения смен для календаря:", err)
		b.answerCallback(callback.ID, "❌ Не удалось загрузить смены")
		return
	}

	keyboard := buildShiftCalendar(month, busyDays)

	edit := tgbotapi.NewEditMessageText(
		chatID,
		messageID,
		"🗓 Календарь смен — "+month.Format("01.2006")+
			"\n\nВыбери нужную дату:",
	)

	edit.ReplyMarkup = &keyboard

	if _, err := b.api.Send(edit); err != nil {
		b.answerCallback(callback.ID, "Ошибка обновления календаря")
		return
	}

	b.answerCallback(callback.ID, "")
}

func (b *Bot) getCalendarBusyDays(
	chatID int64,
	month time.Time,
) (map[int]bool, error) {

	shifts, err := b.shiftService.GetShifts(
		context.Background(),
		chatID,
	)
	if err != nil {
		return nil, err
	}

	busyDays := make(map[int]bool)

	for _, shift := range shifts {
		if shiftInMonth(shift.StartTime, month) {
			busyDays[shift.StartTime.Day()] = true
		}
	}

	return busyDays, nil
}

func (b *Bot) buildCalendarDayText(
	telegramID int64,
	date time.Time,
) (string, error) {
	ctx := context.Background()

	shifts, err := b.shiftService.GetShifts(ctx, telegramID)
	if err != nil {
		return "", err
	}

	var text strings.Builder

	fmt.Fprintf(
		&text,
		"🗓 Смены за %s\n\n",
		date.Format("02.01.2006"),
	)

	count := 0

	for _, shift := range shifts {
		start := shift.StartTime.In(date.Location())
		end := shift.EndTime.In(date.Location())

		// Показываем только смены, начавшиеся
		// в выбранный день.
		if start.Year() != date.Year() ||
			start.Month() != date.Month() ||
			start.Day() != date.Day() {
			continue
		}

		hours, err := service.CalculateHours(shift)
		if err != nil {
			return "", err
		}

		salary, err := service.CalculateSalary(shift)
		if err != nil {
			return "", err
		}

		// Учитываем дополнительные работы.
		extras, err := b.extraRepo.GetExtras(
			ctx,
			telegramID,
			shift.ID,
		)
		if err != nil {
			return "", err
		}

		extraSalary, err := service.CalculateExtraSalary(
			shift,
			extras,
		)
		if err != nil {
			return "", err
		}

		count++

		fmt.Fprintf(
			&text,
			"💼 Смена №%d\n"+
				"🕒 %s — %s\n"+
				"☕ Перерыв: %.0f мин.\n"+
				"⏱ Отработано: %.2f ч.\n"+
				"💰 Заработано: %.2f ₽\n\n",
			count,
			start.Format("02.01 15:04"),
			end.Format("02.01 15:04"),
			shift.Break.Minutes(),
			hours,
			salary+extraSalary,
		)
	}

	if count == 0 {
		text.WriteString("На эту дату смен нет.")
	}

	return text.String(), nil
}

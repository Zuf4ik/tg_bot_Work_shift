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
	if !b.isActiveInlineMessage(chatID, messageID) {
		b.answerCallback(callback.ID, "Календарь устарел")
		return
	}

	data := callback.Data

	if data == "calendar:noop" {
		b.answerCallback(callback.ID, "")
		return
	}

	if strings.HasPrefix(data, "calendar:extra:") {
		shiftIDText := strings.TrimPrefix(data, "calendar:extra:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			b.answerCallback(callback.ID, "Некорректный ID смены")
			return
		}

		if callback.From.ID != chatID {
			b.answerCallback(callback.ID, "Нет доступа")
			return
		}

		b.answerCallback(callback.ID, "")

		// Используем уже существующий обработчик доп. работ.
		b.handleExtraShift(callback, shiftID)
		return
	}

	if strings.HasPrefix(data, "calendar:shift:") {
		// Преобразуем ID смены из строки в число.
		shiftIDText := strings.TrimPrefix(data, "calendar:shift:")

		shiftID, err := strconv.Atoi(shiftIDText)
		if err != nil || shiftID <= 0 {
			b.answerCallback(callback.ID, "Некорректный ID смены")
			return
		}

		// Проверяем, что календарь принадлежит пользователю.
		if callback.From.ID != chatID {
			b.answerCallback(callback.ID, "Нет доступа")
			return
		}

		// Получаем только смену текущего пользователя.
		shift, err := b.shiftService.GetShiftByID(
			context.Background(),
			shiftID,
			callback.From.ID,
		)
		if err != nil || shift == nil {
			b.answerCallback(callback.ID, "Смена не найдена")
			return
		}

		// Используем часовой пояс сохранённой смены.
		loc, err := time.LoadLocation("Asia/Yekaterinburg")
		if err != nil {
			log.Println("ошибка загрузки часового пояса:", err)
			b.answerCallback(callback.ID, "Ошибка обработки времени")
			return
		}

		start := shift.StartTime.In(loc)

		text := formatCalendarShift(*shift, loc)

		keyboard := buildCalendarShiftKeyboard(
			shiftID,
			start,
		)

		edit := tgbotapi.NewEditMessageText(
			chatID,
			messageID,
			text,
		)
		edit.ReplyMarkup = &keyboard

		if _, err := b.api.Send(edit); err != nil {
			log.Println("ошибка открытия карточки смены:", err)
			b.answerCallback(callback.ID, "Ошибка открытия смены")
			return
		}

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

		ctx := context.Background()

		// Получаем смены пользователя.
		shifts, err := b.shiftService.GetShifts(ctx, chatID)
		if err != nil {
			log.Println("ошибка получения смен:", err)
			b.answerCallback(callback.ID, "Не удалось загрузить смены")
			return
		}

		// Оставляем смены только выбранного дня.
		dayShifts := calendarShiftsForDay(shifts, date)

		// Создаём кнопки для каждой найденной смены.
		keyboard := buildCalendarDayKeyboard(date, dayShifts)

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
				"📅 Дата: "+date.Format("02.01.2006")+
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

	ctx := context.Background()

	// Получаем смены пользователя один раз.
	shifts, err := b.shiftService.GetShifts(ctx, chatID)
	if err != nil {
		log.Println("ошибка загрузки смен:", err)
		b.answerCallback(callback.ID, "Не удалось загрузить смены")
		return
	}

	// Рассчитываем часы и зарплату за выбранный месяц.
	hours, salary, err := b.getCalendarMonthTotalsFromShifts(
		ctx,
		chatID,
		month,
		shifts,
	)
	if err != nil {
		log.Println("ошибка расчёта статистики:", err)
		b.answerCallback(callback.ID, "Ошибка расчёта статистики")
		return
	}

	// Определяем дни, в которые есть смены.
	busyDays := calendarBusyDaysFromShifts(shifts, month)

	// Создаём календарь с отметками рабочих дней.
	keyboard := buildShiftCalendar(month, busyDays)

	edit := tgbotapi.NewEditMessageText(
		chatID,
		messageID,
		calendarMonthSummary(month, hours, salary),
	)

	edit.ReplyMarkup = &keyboard

	if _, err := b.api.Send(edit); err != nil {
		b.answerCallback(callback.ID, "Ошибка обновления календаря")
		return
	}

	b.answerCallback(callback.ID, "")
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
	now := time.Now()

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

		incomeLabel := "💰 Заработано"

		if shift.EndTime.After(now) {
			incomeLabel = "📈 Ожидаемый доход"
		}

		fmt.Fprintf(
			&text,
			"💼 Смена\n"+
				"🕒 %s — %s\n"+
				"☕ Перерыв: %.0f мин.\n"+
				"⏱ Продолжительность: %.2f ч.\n"+
				"%s: %.2f ₽\n\n",
			start.Format("02.01 15:04"),
			end.Format("02.01 15:04"),
			shift.Break.Minutes(),
			hours,
			incomeLabel,
			salary+extraSalary,
		)
	}

	if count == 0 {
		text.WriteString("На эту дату смен нет.")
	}

	return text.String(), nil
}

func (b *Bot) getCalendarMonthTotalsFromShifts(
	ctx context.Context,
	telegramID int64,
	month time.Time,
	shifts []model.Shift,
) (float64, float64, error) {
	return sumCalendarMonthTotals(
		shifts,
		month,
		time.Now(),
		func(shiftID int) ([]repository.ExtraType, error) {
			return b.extraRepo.GetExtras(
				ctx,
				telegramID,
				shiftID,
			)
		},
	)
}

func calendarMonthSummary(
	month time.Time,
	hours float64,
	salary float64,
) string {
	return fmt.Sprintf(
		"🗓 Календарь смен — %s\n\n"+
			"⏱ Отработано: %.2f ч.\n"+
			"💰 Заработано: %.2f ₽\n\n"+
			"Выбери нужную дату:",
		month.Format("01.2006"),
		hours,
		salary,
	)
}

func shouldCountCalendarShift(
	shift model.Shift,
	month time.Time,
	now time.Time,
) bool {
	// Смена должна начинаться в выбранном месяце.
	if !shiftInMonth(
		shift.StartTime.In(month.Location()),
		month,
	) {
		return false
	}

	// Незавершённые и будущие смены не учитываем.
	if shift.EndTime.After(now) {
		return false
	}

	return true
}

func calculateCalendarShiftTotals(
	shift model.Shift,
	extras []repository.ExtraType,
) (float64, float64, error) {

	hours, err := service.CalculateHours(shift)
	if err != nil {
		return 0, 0, err
	}

	salary, err := service.CalculateSalary(shift)
	if err != nil {
		return 0, 0, err
	}

	extraSalary, err := service.CalculateExtraSalary(
		shift,
		extras,
	)
	if err != nil {
		return 0, 0, err
	}

	return hours, salary + extraSalary, nil
}

func calendarBusyDaysFromShifts(
	shifts []model.Shift,
	month time.Time,
) map[int]bool {
	busyDays := make(map[int]bool)

	for _, shift := range shifts {
		start := shift.StartTime.In(month.Location())

		if shiftInMonth(start, month) {
			busyDays[start.Day()] = true
		}
	}

	return busyDays
}

func sumCalendarMonthTotals(
	shifts []model.Shift,
	month time.Time,
	now time.Time,
	getExtras func(int) ([]repository.ExtraType, error),
) (float64, float64, error) {
	var totalHours float64
	var totalSalary float64

	for _, shift := range shifts {
		if !shouldCountCalendarShift(shift, month, now) {
			continue
		}

		extras, err := getExtras(shift.ID)
		if err != nil {
			return 0, 0, err
		}

		hours, salary, err := calculateCalendarShiftTotals(
			shift,
			extras,
		)
		if err != nil {
			return 0, 0, err
		}

		totalHours += hours
		totalSalary += salary
	}

	return totalHours, totalSalary, nil
}

func calendarShiftsForDay(
	shifts []model.Shift,
	date time.Time,
) []model.Shift {
	result := make([]model.Shift, 0)

	for _, shift := range shifts {
		start := shift.StartTime.In(date.Location())

		if start.Year() != date.Year() ||
			start.Month() != date.Month() ||
			start.Day() != date.Day() {
			continue
		}

		result = append(result, shift)
	}

	return result
}

func buildCalendarDayKeyboard(
	date time.Time,
	shifts []model.Shift,
) tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton

	// Создаём кнопку для каждой смены.
	for _, shift := range shifts {
		start := shift.StartTime.In(date.Location())
		end := shift.EndTime.In(date.Location())

		label := fmt.Sprintf(
			"💼 %s — %s",
			start.Format("15:04"),
			end.Format("15:04"),
		)

		callback := fmt.Sprintf(
			"calendar:shift:%d",
			shift.ID,
		)

		button := tgbotapi.NewInlineKeyboardButtonData(
			label,
			callback,
		)

		rows = append(
			rows,
			tgbotapi.NewInlineKeyboardRow(button),
		)
	}

	// Добавление новой смены.
	addButton := tgbotapi.NewInlineKeyboardButtonData(
		"➕ Добавить смену",
		"calendar:add:"+date.Format("2006-01-02"),
	)

	rows = append(
		rows,
		tgbotapi.NewInlineKeyboardRow(addButton),
	)

	// Возврат к календарю.
	backButton := tgbotapi.NewInlineKeyboardButtonData(
		"◀️ Назад к календарю",
		"calendar:month:"+date.Format("2006-01"),
	)

	rows = append(
		rows,
		tgbotapi.NewInlineKeyboardRow(backButton),
	)

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func buildCalendarShiftKeyboard(
	shiftID int,
	date time.Time,
) tgbotapi.InlineKeyboardMarkup {
	extraButton := tgbotapi.NewInlineKeyboardButtonData(
		"📦 Дополнительные работы",
		fmt.Sprintf("calendar:extra:%d", shiftID),
	)

	editButton := tgbotapi.NewInlineKeyboardButtonData(
		"✏️ Редактировать смену",
		fmt.Sprintf("edit_shift:%d", shiftID),
	)

	deleteButton := tgbotapi.NewInlineKeyboardButtonData(
		"🗑 Удалить смену",
		fmt.Sprintf("confirm_delete:%d", shiftID),
	)

	backButton := tgbotapi.NewInlineKeyboardButtonData(
		"◀️ Назад к дате",
		"calendar:day:"+date.Format("2006-01-02"),
	)

	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(extraButton),
		tgbotapi.NewInlineKeyboardRow(editButton),
		tgbotapi.NewInlineKeyboardRow(deleteButton),
		tgbotapi.NewInlineKeyboardRow(backButton),
	)
}

func formatCalendarShift(shift model.Shift, loc *time.Location) string {
	start := shift.StartTime.In(loc)
	end := shift.EndTime.In(loc)

	return fmt.Sprintf(
		"💼 Управление сменой\n\n"+
			"📅 Дата: %s\n"+
			"🕒 Время: %s — %s\n"+
			"☕ Перерыв: %.0f мин.\n"+
			"💵 Ставка: %.2f ₽/ч\n\n"+
			"Выбери действие:",
		start.Format("02.01.2006"),
		start.Format("15:04"),
		end.Format("15:04"),
		shift.Break.Minutes(),
		shift.HourlyRate,
	)
}

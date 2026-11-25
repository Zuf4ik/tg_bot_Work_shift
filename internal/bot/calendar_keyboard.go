package bot

import (
	"fmt"
	"time"

	"tg_bot/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func buildShiftCalendarForMode(
	date time.Time,
	busyDays map[int]bool,
	mode CalendarMode,
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
			"calendar:%s:day:%s",
			mode,
			currentDate.Format("2006-01-02"),
		)

		if mode == CalendarModeAdd {
			callback = "calendar:add:" + currentDate.Format("2006-01-02")
		}

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
		fmt.Sprintf("calendar:%s:month:%s", mode, prevMonth.Format("2006-01")),
	)

	nextButton := tgbotapi.NewInlineKeyboardButtonData(
		"▶️",
		fmt.Sprintf("calendar:%s:month:%s", mode, nextMonth.Format("2006-01")),
	)

	// Добавляем строку навигации под календарём.
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		prevButton,
		nextButton,
	))

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

func buildCalendarDayKeyboard(
	date time.Time,
	shifts []model.Shift,
) tgbotapi.InlineKeyboardMarkup {
	return buildCalendarDayKeyboardForMode(date, shifts, CalendarModeShifts)
}

func buildCalendarDayKeyboardForMode(
	date time.Time,
	shifts []model.Shift,
	mode CalendarMode,
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

		var callback string

		switch mode {
		case CalendarModeExtras:
			callback = fmt.Sprintf("calendar:extra:%d", shift.ID)

		case CalendarModeEdit:
			callback = fmt.Sprintf("edit_shift:%d", shift.ID)

		case CalendarModeDelete:
			callback = fmt.Sprintf("confirm_delete:%d", shift.ID)

		default:
			callback = fmt.Sprintf("calendar:shift:%d", shift.ID)
		}
		button := tgbotapi.NewInlineKeyboardButtonData(
			label,
			callback,
		)

		rows = append(
			rows,
			tgbotapi.NewInlineKeyboardRow(button),
		)
	}

	if mode == CalendarModeShifts {
		addButton := tgbotapi.NewInlineKeyboardButtonData(
			"➕ Добавить смену",
			"calendar:add:"+date.Format("2006-01-02"),
		)

		rows = append(
			rows,
			tgbotapi.NewInlineKeyboardRow(addButton),
		)
	}

	// Возврат к календарю.
	backButton := tgbotapi.NewInlineKeyboardButtonData(
		"◀️ Назад к календарю",
		fmt.Sprintf(
			"calendar:%s:month:%s",
			mode,
			date.Format("2006-01"),
		),
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

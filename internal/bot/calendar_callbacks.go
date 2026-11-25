package bot

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

	// Календарь и сведения о сменах доступны только владельцу чата.
	if callback.From == nil || callback.From.ID != chatID {
		b.answerCallback(callback.ID, "Нет доступа")
		return
	}

	data := callback.Data

	mode := CalendarModeShifts

	switch {
	case strings.HasPrefix(data, "calendar:extras:"):
		mode = CalendarModeExtras
		data = strings.TrimPrefix(data, "calendar:extras:")
		data = "calendar:" + data

	case strings.HasPrefix(data, "calendar:add:day:"),
		strings.HasPrefix(data, "calendar:add:month:"):
		mode = CalendarModeAdd
		data = strings.TrimPrefix(data, "calendar:add:")
		data = "calendar:" + data

	case strings.HasPrefix(data, "calendar:edit:"):
		mode = CalendarModeEdit
		data = "calendar:" + strings.TrimPrefix(data, "calendar:edit:")

	case strings.HasPrefix(data, "calendar:delete:"):
		mode = CalendarModeDelete
		data = "calendar:" + strings.TrimPrefix(data, "calendar:delete:")

	case strings.HasPrefix(data, "calendar:shifts:"):
		mode = CalendarModeShifts
		data = "calendar:" + strings.TrimPrefix(data, "calendar:shifts:")
	}

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
		keyboard := buildCalendarDayKeyboardForMode(date, dayShifts, mode)

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
	stats, err := b.calculateCalendarMonthStats(
		ctx,
		callback.From.ID,
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
	keyboard := buildShiftCalendarForMode(month, busyDays, mode)

	edit := tgbotapi.NewEditMessageText(
		chatID,
		messageID,
		calendarMonthSummary(month, stats),
	)

	edit.ReplyMarkup = &keyboard

	if _, err := b.api.Send(edit); err != nil {
		b.answerCallback(callback.ID, "Ошибка обновления календаря")
		return
	}

	b.answerCallback(callback.ID, "")
}

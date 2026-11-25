package bot

import (
	"context"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CalendarMode string

type CalendarMonthStats struct {
	WorkedHours  float64
	PlannedHours float64
	Salary       float64
	NormHours    float64
	Remaining    float64
	Bonus        float64
	HasNorm      bool
}

const (
	CalendarModeShifts CalendarMode = "shifts"
	CalendarModeExtras CalendarMode = "extras"
	CalendarModeAdd    CalendarMode = "add"
	CalendarModeEdit   CalendarMode = "edit"
	CalendarModeDelete CalendarMode = "delete"
)

func (b *Bot) handleShiftCalendar(
	message *tgbotapi.Message,
) {
	b.handleShiftCalendarForMode(message, CalendarModeShifts)
}

func (b *Bot) handleShiftCalendarForMode(
	message *tgbotapi.Message,
	mode CalendarMode,
) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		log.Println("ошибка загрузки часового пояса:", err)
		return
	}

	month := time.Now().In(loc)

	ctx := context.Background()

	// Получаем все смены пользователя один раз.
	shifts, err := b.shiftService.GetShifts(
		ctx,
		message.Chat.ID,
	)
	if err != nil {
		log.Println("ошибка загрузки смен:", err)
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось загрузить смены.",
		))
		return
	}

	// Рассчитываем статистику из полученных смен.
	stats, err := b.calculateCalendarMonthStats(
		ctx,
		message.From.ID,
		month,
		shifts,
	)
	if err != nil {
		log.Println("ошибка расчёта статистики:", err)
		b.api.Send(tgbotapi.NewMessage(
			message.Chat.ID,
			"❌ Не удалось рассчитать статистику месяца.",
		))
		return
	}

	msg := tgbotapi.NewMessage(
		message.Chat.ID,
		calendarMonthSummary(month, stats),
	)

	busyDays := calendarBusyDaysFromShifts(shifts, month)

	msg.ReplyMarkup = buildShiftCalendarForMode(month, busyDays, mode)

	sent, err := b.api.Send(msg)
	if err != nil {
		log.Println("ошибка отправки календаря:", err)
		return
	}

	b.activeInlineMessages[message.Chat.ID] = sent.MessageID
}

func calendarMonthStart(date time.Time) time.Time {
	return time.Date(
		date.Year(),
		date.Month(),
		1,
		0, 0, 0, 0,
		date.Location(),
	)
}

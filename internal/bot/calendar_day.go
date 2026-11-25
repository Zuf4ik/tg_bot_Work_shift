package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"tg_bot/model"
	"tg_bot/service"
)

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

package bot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tg_bot/internal/repository"
	"tg_bot/model"
	"tg_bot/service"

	"github.com/jackc/pgx/v5"
)

func calculateCalendarPlannedHours(
	shifts []model.Shift,
	month time.Time,
	now time.Time,
) (float64, error) {
	var plannedHours float64

	for _, shift := range shifts {
		start := shift.StartTime.In(month.Location())

		// Учитываем только смены выбранного месяца.
		if !shiftInMonth(start, month) {
			continue
		}

		// Завершённые смены уже входят в WorkedHours.
		if !shift.EndTime.After(now) {
			continue
		}

		hours, err := service.CalculateHours(shift)
		if err != nil {
			return 0, err
		}

		plannedHours += hours
	}

	return plannedHours, nil
}

func (b *Bot) calculateCalendarMonthStats(
	ctx context.Context,
	telegramID int64,
	month time.Time,
	shifts []model.Shift,
) (CalendarMonthStats, error) {

	// 1. Отработанные часы и заработанная зарплата.
	workedHours, salary, err := b.getCalendarMonthTotalsFromShifts(
		ctx,
		telegramID,
		month,
		shifts,
	)
	if err != nil {
		return CalendarMonthStats{}, err
	}

	// 2. Запланированные часы.
	plannedHours, err := calculateCalendarPlannedHours(
		shifts,
		month,
		time.Now(),
	)
	if err != nil {
		return CalendarMonthStats{}, err
	}

	stats := CalendarMonthStats{
		WorkedHours:  workedHours,
		PlannedHours: plannedHours,
		Salary:       salary,
	}

	// 3. Месячная норма и премия.
	bonusResult, err := b.salaryService.CalculateMonthlyBonus(
		ctx,
		month.Year(),
		int(month.Month()),
		workedHours,
	)

	// Если норма ещё не внесена в БД,
	// календарь всё равно должен работать.
	if errors.Is(err, pgx.ErrNoRows) {
		return stats, nil
	}

	if err != nil {
		return CalendarMonthStats{}, err
	}

	stats.NormHours = bonusResult.NormHours
	stats.Remaining = bonusResult.Remaining
	stats.Bonus = bonusResult.Bonus
	stats.HasNorm = true

	return stats, nil
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
	stats CalendarMonthStats,
) string {
	summary := fmt.Sprintf(
		"🗓 Календарь смен — %s\n\n"+
			"✅ Отработано: %.2f ч.\n"+
			"📅 Запланировано: %.2f ч.\n",
		month.Format("01.2006"),
		stats.WorkedHours,
		stats.PlannedHours,
	)

	if stats.HasNorm {
		progress := stats.WorkedHours / stats.NormHours * 100

		summary += fmt.Sprintf(
			"🎯 Норма месяца: %.2f ч.\n"+
				"⌛ Осталось до нормы: %.2f ч.\n"+
				"📊 Выполнено: %.1f%%\n"+
				"🏆 Премия: %.2f ₽\n",
			stats.NormHours,
			stats.Remaining,
			progress,
			stats.Bonus,
		)
	} else {
		summary += "🎯 Норма месяца: не указана\n"
	}

	summary += fmt.Sprintf(
		"\n💰 Заработано: %.2f ₽\n\n"+
			"Выбери нужную дату:",
		stats.Salary,
	)

	return summary
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

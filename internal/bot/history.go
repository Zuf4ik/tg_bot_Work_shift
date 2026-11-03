package bot

import "time"

// monthStart возвращает начало месяца для указанной даты.
func monthStart(date time.Time) time.Time {
	return time.Date(
		date.Year(),
		date.Month(),
		1,
		0, 0, 0, 0,
		date.Location(),
	)
}

// shiftInMonth проверяет, относится ли смена к выбранному месяцу.
// Смена учитывается по дате её начала.
func shiftInMonth(shiftStart, month time.Time) bool {
	start := monthStart(month)
	end := start.AddDate(0, 1, 0)

	return !shiftStart.Before(start) && shiftStart.Before(end)
}

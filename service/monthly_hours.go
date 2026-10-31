package service

import (
	"fmt"
	"time"

	"tg_bot/model"
)

func CalculateMonthlyHours(
	shifts []model.Shift,
	year int,
	month time.Month,
) (float64, error) {
	if year < 2020 || year > 2036 {
		return 0, fmt.Errorf("некорректный год")
	}

	if month < time.January || month > time.December {
		return 0, fmt.Errorf("некорректный месяц")
	}

	var totalHours float64

	for _, shift := range shifts {
		start := shift.StartTime

		// Пока относим смену к месяцу её начала,
		// как в существующем отчёте.
		if start.Year() != year || start.Month() != month {
			continue
		}

		hours, err := CalculateHours(shift)
		if err != nil {
			return 0, fmt.Errorf(
				"ошибка расчёта смены %d: %w",
				shift.ID,
				err,
			)
		}

		totalHours += hours
	}

	return totalHours, nil
}

package service

import (
	"fmt"

	"tg_bot/internal/repository"
	"tg_bot/model"
)

// CalculateExtraSalary рассчитывает доплату сверх
// основной зарплаты за смену.
func CalculateExtraSalary(
	shift model.Shift,
	extras []repository.ExtraType,
) (float64, error) {
	hours, err := CalculateHours(shift)
	if err != nil {
		return 0, err
	}

	if _, err := CalculateSalary(shift); err != nil {
		return 0, err
	}

	var extraSalary float64
	seen := make(map[repository.ExtraType]bool)

	for _, extra := range extras {
		if seen[extra] {
			return 0, fmt.Errorf(
				"дополнительная работа %s указана повторно",
				extra,
			)
		}
		seen[extra] = true

		switch extra {
		case repository.ExtraUnloading:
			extraSalary += 2 * shift.HourlyRate

		case repository.ExtraTrash:
			extraSalary += 0.25 * shift.HourlyRate

		case repository.ExtraDelivery:
			// Половина оплачиваемых часов получает
			// дополнительные 100% ставки.
			extraSalary += (hours / 2) * shift.HourlyRate

		default:
			return 0, fmt.Errorf(
				"неизвестный тип дополнительной работы: %s",
				extra,
			)
		}
	}

	return extraSalary, nil
}

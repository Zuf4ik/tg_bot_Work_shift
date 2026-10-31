package service

import (
	"context"
	"fmt"
	"time"

	"tg_bot/model"
)

// CalendarNormProvider позволяет получать месячную норму.
type CalendarNormProvider interface {
	GetMonthlyNorm(ctx context.Context, year int, month int) (float64, error)
}

type SalaryService struct {
	calendar CalendarNormProvider
}

func NewSalaryService(calendar CalendarNormProvider) *SalaryService {
	return &SalaryService{
		calendar: calendar,
	}
}

// MonthlyBonusResult содержит результат расчёта премии.
type MonthlyBonusResult struct {
	Year        int
	Month       int
	WorkedHours float64
	NormHours   float64
	Remaining   float64
	Bonus       float64
}

// CalculateMonthlyBonus рассчитывает премию за месяц.
func (s *SalaryService) CalculateMonthlyBonus(
	ctx context.Context,
	year int,
	month int,
	workedHours float64,
) (MonthlyBonusResult, error) {
	if workedHours < 0 {
		return MonthlyBonusResult{}, fmt.Errorf(
			"отработанные часы не могут быть отрицательными",
		)
	}

	if year < 2020 || year > 2036 || month < 1 || month > 12 {
		return MonthlyBonusResult{}, fmt.Errorf(
			"некорректный месяц или год",
		)
	}

	norm, err := s.calendar.GetMonthlyNorm(ctx, year, month)
	if err != nil {
		return MonthlyBonusResult{}, err
	}

	if norm <= 0 {
		return MonthlyBonusResult{}, fmt.Errorf(
			"норма часов должна быть положительной",
		)
	}

	remaining := norm - workedHours
	if remaining < 0 {
		remaining = 0
	}

	return MonthlyBonusResult{
		Year:        year,
		Month:       month,
		WorkedHours: workedHours,
		NormHours:   norm,
		Remaining:   remaining,
		Bonus:       CalculateBonus(workedHours, norm),
	}, nil
}

func (s *SalaryService) CalculateBonusFromShifts(
	ctx context.Context,
	year int,
	month time.Month,
	shifts []model.Shift,
) (MonthlyBonusResult, error) {
	hours, err := CalculateMonthlyHours(shifts, year, month)
	if err != nil {
		return MonthlyBonusResult{}, err
	}

	return s.CalculateMonthlyBonus(
		ctx,
		year,
		int(month),
		hours,
	)
}

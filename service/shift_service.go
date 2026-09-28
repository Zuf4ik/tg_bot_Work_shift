package service

import (
	"context"
	"errors"
	"time"

	"tg_bot/internal/repository"
	"tg_bot/model"
)

func CalculateHours(shift model.Shift) (float64, error) {
	if shift.EndTime.Before(shift.StartTime) {
		return 0, errors.New("время окончания раньше времени начала")
	}

	duration := shift.EndTime.Sub(shift.StartTime) - shift.Break

	if duration < 0 {
		return 0, errors.New("перерыв больше продолжительности смены")
	}

	return duration.Hours(), nil
}

func CalculateSalary(shift model.Shift) (float64, error) {
	if shift.EndTime.Before(shift.StartTime) {
		return 0, errors.New("время окончания раньше времени начала")
	}

	if shift.HourlyRate < 0 {
		return 0, errors.New("ставка не может быть отрицательной")
	}

	if shift.Break < 0 {
		return 0, errors.New("перерыв не может быть отрицательным")
	}

	totalDuration := shift.EndTime.Sub(shift.StartTime)

	if shift.Break > totalDuration {
		return 0, errors.New("перерыв больше продолжительности смены")
	}

	// Считаем зарплату до вычета перерыва.
	var salary float64

	current := shift.StartTime

	for current.Before(shift.EndTime) {
		next := current.Add(time.Minute)

		if next.After(shift.EndTime) {
			next = shift.EndTime
		}

		minuteHours := next.Sub(current).Hours()

		hour := current.Hour()

		// Ночное время: 22:00–06:00.
		isNight := hour >= 22 || hour < 6

		coefficient := 1.0

		if isNight {
			coefficient = 1.3
		}

		salary += minuteHours * shift.HourlyRate * coefficient

		current = next
	}

	// Пока мы не храним время начала перерыва,
	// поэтому считаем перерыв по базовой ставке.
	salary -= shift.Break.Hours() * shift.HourlyRate

	if salary < 0 {
		return 0, errors.New("расчёт зарплаты дал отрицательный результат")
	}

	return salary, nil
}

type ShiftService struct {
	repo *repository.ShiftRepository
}

func NewShiftService(repo *repository.ShiftRepository) *ShiftService {
	return &ShiftService{
		repo: repo,
	}
}

func (s *ShiftService) CreateShift(ctx context.Context, shift model.Shift) (int, error) {
	_, err := CalculateHours(shift)
	if err != nil {
		return 0, err
	}

	_, err = CalculateSalary(shift)
	if err != nil {
		return 0, err
	}

	return s.repo.CreateShift(ctx, shift)
}

func (s *ShiftService) GetShifts(ctx context.Context) ([]model.Shift, error) {
	return s.repo.GetShifts(ctx)
}

func (s *ShiftService) UpdateShift(ctx context.Context, id int, shift model.Shift) error {
	_, err := CalculateHours(shift)
	if err != nil {
		return err
	}

	_, err = CalculateSalary(shift)
	if err != nil {
		return err
	}

	return s.repo.UpdateShift(ctx, id, shift)
}

func (s *ShiftService) DeleteShift(ctx context.Context, id int) error {
	return s.repo.DeleteShift(ctx, id)
}

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

	if shift.HourlyRate > 10000 {
		return 0, errors.New("почасовая ставка не может превышать 10000")
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

// validateNoOverlap запрещает пересечение смен одного пользователя.
// Смены, которые соприкасаются границами (например, 07:00–16:00 и 16:00–20:00), допустимы.
func (s *ShiftService) validateNoOverlap(ctx context.Context, shift model.Shift, excludeID int) error {
	shifts, err := s.repo.GetShifts(ctx, shift.TelegramID)
	if err != nil {
		return err
	}

	for _, existing := range shifts {
		if excludeID != 0 && existing.ID == excludeID {
			continue
		}
		// Два интервала пересекаются, если начало каждого раньше окончания другого.
		if shift.StartTime.Before(existing.EndTime) && shift.EndTime.After(existing.StartTime) {
			return errors.New("эта смена пересекается с уже сохранённой сменой " +
				existing.StartTime.Format("02.01.2006 15:04") + "–" +
				existing.EndTime.Format("02.01.2006 15:04"))
		}
	}
	return nil
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
	if err = s.validateNoOverlap(ctx, shift, 0); err != nil {
		return 0, err
	}

	return s.repo.CreateShift(ctx, shift)
}

func (s *ShiftService) GetShifts(
	ctx context.Context,
	telegramID int64,
) ([]model.Shift, error) {
	return s.repo.GetShifts(ctx, telegramID)
}

func (s *ShiftService) GetShiftByID(
	ctx context.Context,
	id int,
	telegramID int64,
) (*model.Shift, error) {
	return s.repo.GetShiftByID(ctx, id, telegramID)
}

func (s *ShiftService) UpdateShift(
	ctx context.Context,
	id int,
	telegramID int64,
	shift model.Shift,
) error {
	_, err := CalculateHours(shift)
	if err != nil {
		return err
	}

	_, err = CalculateSalary(shift)
	if err != nil {
		return err
	}

	if err = s.validateNoOverlap(ctx, shift, id); err != nil {
		return err
	}

	return s.repo.UpdateShift(ctx, id, telegramID, shift)
}

func (s *ShiftService) DeleteShift(
	ctx context.Context,
	id int,
	telegramID int64,
) error {
	return s.repo.DeleteShift(ctx, id, telegramID)
}

package service

import (
	"context"
	"errors"
	"testing"
)

type mockCalendar struct {
	norm float64
	err  error
}

func (m mockCalendar) GetMonthlyNorm(
	ctx context.Context,
	year int,
	month int,
) (float64, error) {
	return m.norm, m.err
}

func TestCalculateMonthlyBonus(t *testing.T) {
	tests := []struct {
		name          string
		workedHours   float64
		norm          float64
		expectedBonus float64
		expectedLeft  float64
	}{
		{"Меньше 80 часов", 70, 176, 0, 106},
		{"80 часов", 80, 176, 1500, 96},
		{"120 часов", 120, 176, 3000, 56},
		{"140 часов", 140, 176, 8000, 36},
		{"Выполнена норма", 176, 176, 15000, 0},
		{"Сверх нормы", 180, 176, 15000, 0},
		{"Короткий месяц", 120, 120, 15000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSalaryService(mockCalendar{norm: tt.norm})

			result, err := s.CalculateMonthlyBonus(
				context.Background(),
				2026,
				10,
				tt.workedHours,
			)
			if err != nil {
				t.Fatal(err)
			}

			if result.Bonus != tt.expectedBonus {
				t.Errorf(
					"премия: ожидалось %.2f, получено %.2f",
					tt.expectedBonus,
					result.Bonus,
				)
			}

			if result.Remaining != tt.expectedLeft {
				t.Errorf(
					"остаток: ожидалось %.2f, получено %.2f",
					tt.expectedLeft,
					result.Remaining,
				)
			}
		})
	}
}

func TestCalculateMonthlyBonusErrors(t *testing.T) {
	tests := []struct {
		name  string
		year  int
		month int
		hours float64
		norm  float64
		err   error
	}{
		{"Отрицательные часы", 2026, 10, -1, 176, nil},
		{"Неверный месяц", 2026, 13, 100, 176, nil},
		{"Неверный год", 2019, 10, 100, 176, nil},
		{"Нулевая норма", 2026, 10, 100, 0, nil},
		{"Ошибка базы", 2026, 10, 100, 176, errors.New("ошибка БД")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSalaryService(mockCalendar{
				norm: tt.norm,
				err:  tt.err,
			})

			_, err := s.CalculateMonthlyBonus(
				context.Background(),
				tt.year,
				tt.month,
				tt.hours,
			)

			if err == nil {
				t.Error("ожидалась ошибка, но её нет")
			}
		})
	}
}

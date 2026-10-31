package service

import (
	"math"
	"testing"
	"time"

	"tg_bot/model"
)

func TestCalculateMonthlyHours(t *testing.T) {
	tests := []struct {
		name     string
		shifts   []model.Shift
		year     int
		month    time.Month
		expected float64
	}{
		{
			name: "Две смены за октябрь",
			shifts: []model.Shift{
				{
					StartTime: time.Date(2026, 10, 10, 10, 0, 0, 0, time.Local),
					EndTime:   time.Date(2026, 10, 10, 18, 0, 0, 0, time.Local),
					Break:     time.Hour,
				},
				{
					StartTime: time.Date(2026, 10, 11, 9, 0, 0, 0, time.Local),
					EndTime:   time.Date(2026, 10, 11, 17, 0, 0, 0, time.Local),
					Break:     30 * time.Minute,
				},
			},
			year:     2026,
			month:    time.October,
			expected: 14.5,
		},
		{
			name: "Смена другого месяца не учитывается",
			shifts: []model.Shift{
				{
					StartTime: time.Date(2026, 9, 30, 10, 0, 0, 0, time.Local),
					EndTime:   time.Date(2026, 9, 30, 18, 0, 0, 0, time.Local),
				},
			},
			year:     2026,
			month:    time.October,
			expected: 0,
		},
		{
			name:     "Нет смен",
			shifts:   []model.Shift{},
			year:     2026,
			month:    time.October,
			expected: 0,
		},
		{
			name: "Ночная смена через полночь",
			shifts: []model.Shift{
				{
					StartTime: time.Date(2026, 10, 31, 22, 0, 0, 0, time.Local),
					EndTime:   time.Date(2026, 11, 1, 7, 0, 0, 0, time.Local),
				},
			},
			year:     2026,
			month:    time.October,
			expected: 9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateMonthlyHours(
				tt.shifts,
				tt.year,
				tt.month,
			)
			if err != nil {
				t.Fatal(err)
			}

			if math.Abs(got-tt.expected) > 0.001 {
				t.Errorf(
					"ожидалось %.2f ч, получено %.2f ч",
					tt.expected,
					got,
				)
			}
		})
	}
}

func TestCalculateMonthlyHoursInvalidMonth(t *testing.T) {
	_, err := CalculateMonthlyHours(nil, 2026, 13)

	if err == nil {
		t.Fatal("ожидалась ошибка для некорректного месяца")
	}
}

package bot

import (
	"math"
	"testing"
	"tg_bot/internal/repository"
	"tg_bot/model"
	"time"
)

func TestShouldCountCalendarShift(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	at := func(month time.Month, day, hour int) time.Time {
		return time.Date(2026, month, day, hour, 0, 0, 0, loc)
	}

	month := at(time.November, 1, 0)
	now := at(time.November, 13, 15)

	tests := []struct {
		name  string
		shift model.Shift
		want  bool
	}{
		{
			name: "Завершённая смена",
			shift: model.Shift{
				StartTime: at(time.November, 10, 9),
				EndTime:   at(time.November, 10, 16),
			},
			want: true,
		},
		{
			name: "Будущая смена",
			shift: model.Shift{
				StartTime: at(time.November, 20, 9),
				EndTime:   at(time.November, 20, 16),
			},
			want: false,
		},
		{
			name: "Смена ещё идёт",
			shift: model.Shift{
				StartTime: at(time.November, 13, 14),
				EndTime:   at(time.November, 13, 18),
			},
			want: false,
		},
		{
			name: "Смена началась в октябре",
			shift: model.Shift{
				StartTime: at(time.October, 31, 22),
				EndTime:   at(time.November, 1, 7),
			},
			want: false,
		},
		{
			name: "Ночная смена через полночь",
			shift: model.Shift{
				StartTime: at(time.November, 11, 22),
				EndTime:   at(time.November, 12, 7),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldCountCalendarShift(
				tt.shift,
				month,
				now,
			)

			if got != tt.want {
				t.Errorf(
					"ожидалось %t, получено %t",
					tt.want,
					got,
				)
			}
		})
	}
}

func TestCalculateCalendarShiftTotals(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	shift := model.Shift{
		StartTime:  time.Date(2026, 11, 10, 9, 0, 0, 0, loc),
		EndTime:    time.Date(2026, 11, 10, 16, 0, 0, 0, loc),
		Break:      time.Hour,
		HourlyRate: 250,
	}

	extras := []repository.ExtraType{
		repository.ExtraUnloading,
	}

	hours, salary, err := calculateCalendarShiftTotals(
		shift,
		extras,
	)
	if err != nil {
		t.Fatal(err)
	}

	expectedHours := 6.0
	expectedSalary := 2000.0

	if math.Abs(hours-expectedHours) > 0.001 {
		t.Errorf(
			"ожидалось %.2f часов, получено %.2f",
			expectedHours,
			hours,
		)
	}

	if math.Abs(salary-expectedSalary) > 0.001 {
		t.Errorf(
			"ожидалось %.2f ₽, получено %.2f ₽",
			expectedSalary,
			salary,
		)
	}
}

func TestCalculateCalendarNightDelivery(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	shift := model.Shift{
		StartTime:  time.Date(2026, 11, 10, 22, 0, 0, 0, loc),
		EndTime:    time.Date(2026, 11, 11, 7, 0, 0, 0, loc),
		Break:      0,
		HourlyRate: 250,
	}

	extras := []repository.ExtraType{
		repository.ExtraDelivery,
	}

	hours, earned, err := calculateCalendarShiftTotals(
		shift,
		extras,
	)
	if err != nil {
		t.Fatal(err)
	}

	expectedHours := 9.0
	expectedSalary := 4000.0

	if math.Abs(hours-expectedHours) > 0.001 {
		t.Errorf(
			"ожидалось %.2f ч., получено %.2f ч.",
			expectedHours,
			hours,
		)
	}

	if math.Abs(earned-expectedSalary) > 0.001 {
		t.Errorf(
			"ожидалось %.2f ₽, получено %.2f ₽",
			expectedSalary,
			earned,
		)
	}
}

func TestSumCalendarMonthTotals(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	at := func(month time.Month, day, hour int) time.Time {
		return time.Date(2026, month, day, hour, 0, 0, 0, loc)
	}

	shifts := []model.Shift{
		{
			ID:         1,
			StartTime:  at(time.November, 10, 9),
			EndTime:    at(time.November, 10, 16),
			Break:      time.Hour,
			HourlyRate: 250,
		},
		{
			ID:         2,
			StartTime:  at(time.November, 11, 22),
			EndTime:    at(time.November, 12, 7),
			HourlyRate: 250,
		},
		{
			ID:         3,
			StartTime:  at(time.November, 20, 9),
			EndTime:    at(time.November, 20, 16),
			HourlyRate: 250,
		},
		{
			ID:         4,
			StartTime:  at(time.October, 31, 9),
			EndTime:    at(time.October, 31, 16),
			HourlyRate: 250,
		},
	}

	// Имитируем данные из PostgreSQL.
	extras := map[int][]repository.ExtraType{
		1: {repository.ExtraUnloading},
		2: {repository.ExtraDelivery},
	}

	now := at(time.November, 13, 15)
	month := at(time.November, 1, 0)

	extrasCalls := 0

	hours, salary, err := sumCalendarMonthTotals(
		shifts,
		month,
		now,
		func(shiftID int) ([]repository.ExtraType, error) {
			extrasCalls++
			return extras[shiftID], nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if math.Abs(hours-15.0) > 0.001 {
		t.Errorf("ожидалось 15 часов, получено %.2f", hours)
	}

	if math.Abs(salary-6000.0) > 0.001 {
		t.Errorf("ожидалось 6000 ₽, получено %.2f ₽", salary)
	}

	if extrasCalls != 2 {
		t.Errorf(
			"ожидалось 2 обращения за доп. работами, получено %d",
			extrasCalls,
		)
	}
}

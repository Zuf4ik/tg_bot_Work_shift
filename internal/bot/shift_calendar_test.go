package bot

import (
	"math"
	"reflect"
	"strings"
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

func TestCalendarShiftsForDay(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	makeTime := func(day, hour int) time.Time {
		return time.Date(
			2026, time.November, day,
			hour, 0, 0, 0, loc,
		)
	}

	shifts := []model.Shift{
		{
			ID:        1,
			StartTime: makeTime(12, 9),
			EndTime:   makeTime(12, 16),
		},
		{
			ID:        2,
			StartTime: makeTime(12, 18),
			EndTime:   makeTime(12, 23),
		},
		{
			ID:        3,
			StartTime: makeTime(11, 22),
			EndTime:   makeTime(12, 7),
		},
		{
			ID:        4,
			StartTime: makeTime(13, 9),
			EndTime:   makeTime(13, 16),
		},
	}

	date := makeTime(12, 0)

	result := calendarShiftsForDay(shifts, date)

	if len(result) != 2 {
		t.Fatalf(
			"ожидалось 2 смены, получено %d",
			len(result),
		)
	}

	if result[0].ID != 1 || result[1].ID != 2 {
		t.Errorf(
			"неверные смены: получены ID %d и %d",
			result[0].ID,
			result[1].ID,
		)
	}
}

func TestBuildCalendarDayKeyboard(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	date := time.Date(2026, 11, 12, 0, 0, 0, 0, loc)

	shifts := []model.Shift{
		{
			ID:        10,
			StartTime: date.Add(9 * time.Hour),
			EndTime:   date.Add(16 * time.Hour),
		},
		{
			ID:        20,
			StartTime: date.Add(18 * time.Hour),
			EndTime:   date.Add(23 * time.Hour),
		},
	}

	keyboard := buildCalendarDayKeyboard(date, shifts)

	expectedCallbacks := []string{
		"calendar:shift:10",
		"calendar:shift:20",
		"calendar:add:2026-11-12",
		"calendar:shifts:month:2026-11",
	}

	if len(keyboard.InlineKeyboard) != len(expectedCallbacks) {
		t.Fatalf(
			"ожидалось %d строк, получено %d",
			len(expectedCallbacks),
			len(keyboard.InlineKeyboard),
		)
	}

	for i, expected := range expectedCallbacks {
		row := keyboard.InlineKeyboard[i]

		if len(row) != 1 {
			t.Fatalf("строка %d: ожидалась одна кнопка", i)
		}

		if row[0].CallbackData == nil {
			t.Fatalf("строка %d: отсутствует callback", i)
		}

		if actual := *row[0].CallbackData; actual != expected {
			t.Errorf(
				"строка %d: ожидалось %q, получено %q",
				i,
				expected,
				actual,
			)
		}
	}
}

func TestBuildCalendarShiftKeyboard(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	date := time.Date(
		2026, time.November, 12,
		9, 0, 0, 0, loc,
	)

	keyboard := buildCalendarShiftKeyboard(125, date)

	tests := []struct {
		name     string
		callback string
	}{
		{
			name:     "Дополнительные работы",
			callback: "calendar:extra:125",
		},
		{
			name:     "Редактирование",
			callback: "edit_shift:125",
		},
		{
			name:     "Удаление",
			callback: "confirm_delete:125",
		},
		{
			name:     "Возврат к дате",
			callback: "calendar:day:2026-11-12",
		},
	}

	if len(keyboard.InlineKeyboard) != len(tests) {
		t.Fatalf(
			"ожидалось %d строк, получено %d",
			len(tests),
			len(keyboard.InlineKeyboard),
		)
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			row := keyboard.InlineKeyboard[i]

			if len(row) != 1 {
				t.Fatalf("ожидалась одна кнопка, получено %d", len(row))
			}

			if row[0].CallbackData == nil {
				t.Fatal("у кнопки отсутствует callback")
			}

			if actual := *row[0].CallbackData; actual != tt.callback {
				t.Errorf(
					"ожидалось %q, получено %q",
					tt.callback,
					actual,
				)
			}
		})
	}
}

func TestFormatCalendarShift(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	shift := model.Shift{
		ID: 125,
		StartTime: time.Date(
			2026, 11, 12, 4, 0, 0, 0, time.UTC,
		),
		EndTime: time.Date(
			2026, 11, 12, 11, 0, 0, 0, time.UTC,
		),
		Break:      time.Hour,
		HourlyRate: 250,
	}

	result := formatCalendarShift(shift, loc)

	expected := []string{
		"12.11.2026",
		"09:00 — 16:00",
		"Перерыв: 60 мин.",
		"Ставка: 250.00 ₽/ч",
	}

	for _, value := range expected {
		if !strings.Contains(result, value) {
			t.Errorf(
				"в карточке отсутствует %q\nПолученный текст:\n%s",
				value,
				result,
			)
		}
	}
}

func TestBuildCalendarDayKeyboardExtras(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	date := time.Date(2026, 11, 20, 0, 0, 0, 0, loc)

	shifts := []model.Shift{
		{
			ID:        125,
			StartTime: date.Add(16 * time.Hour),
			EndTime:   date.Add(24 * time.Hour),
		},
	}

	keyboard := buildCalendarDayKeyboardForMode(
		date,
		shifts,
		CalendarModeExtras,
	)

	var callbacks []string

	for _, row := range keyboard.InlineKeyboard {
		for _, button := range row {
			if button.CallbackData != nil {
				callbacks = append(callbacks, *button.CallbackData)
			}
		}
	}

	expected := []string{
		"calendar:extra:125",
		"calendar:extras:month:2026-11",
	}

	if !reflect.DeepEqual(callbacks, expected) {
		t.Fatalf(
			"ожидались callback %v, получены %v",
			expected,
			callbacks,
		)
	}
}

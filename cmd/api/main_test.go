package main

import (
	"testing"
	"time"
)

func TestParseShiftTime(t *testing.T) {
	input := "2026-11-12T09:00:00"

	result, err := parseShiftTime(input)
	if err != nil {
		t.Fatalf("Ошибка преобразования времени: %v", err)
	}

	// 09:00 в Уфе соответствует 04:00 UTC.
	expected := time.Date(
		2026, 11, 12,
		4, 0, 0, 0,
		time.UTC,
	)

	if !result.Equal(expected) {
		t.Errorf(
			"Ожидалось %v, получено %v",
			expected,
			result,
		)
	}

	if result.Format("15:04") != "09:00" {
		t.Errorf(
			"Ожидалось 09:00, получено %s",
			result.Format("15:04"),
		)
	}
}

func TestValidateShiftInput(t *testing.T) {
	valid := createShiftRequest{TelegramID: 123, BreakMinutes: 30, HourlyRate: 250}
	if err := validateShiftInput(valid, true); err != nil {
		t.Fatalf("корректные данные отклонены: %v", err)
	}
	invalid := []struct {
		name string
		req  createShiftRequest
	}{
		{"Отсутствует TelegramID", createShiftRequest{BreakMinutes: 30, HourlyRate: 250}},
		{"Отрицательный перерыв", createShiftRequest{TelegramID: 123, BreakMinutes: -1, HourlyRate: 250}},
		{"Слишком длинный перерыв", createShiftRequest{TelegramID: 123, BreakMinutes: 361, HourlyRate: 250}},
		{"Отсутствует ставка", createShiftRequest{TelegramID: 123, BreakMinutes: 30}},
		{"Ставка выше лимита", createShiftRequest{TelegramID: 123, BreakMinutes: 30, HourlyRate: 10001}},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			if err := validateShiftInput(test.req, true); err == nil {
				t.Fatal("ожидалась ошибка валидации")
			}
		})
	}
	if err := validateShiftInput(createShiftRequest{BreakMinutes: 0, HourlyRate: 250}, false); err != nil {
		t.Fatalf("PUT должен брать telegram_id из параметра запроса: %v", err)
	}
}

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

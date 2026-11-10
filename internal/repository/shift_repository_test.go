package repository

import (
	"testing"
	"time"
)

func TestLocalTime(t *testing.T) {
	ufa, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	// Пользователь ввёл смену 09:00–16:00 по Уфе.
	start := time.Date(2026, 11, 10, 9, 0, 0, 0, ufa)
	end := time.Date(2026, 11, 10, 16, 0, 0, 0, ufa)

	// Имитируем чтение времени из БД в UTC.
	savedStart := start.UTC()
	savedEnd := end.UTC()

	restoredStart := localTime(savedStart)
	restoredEnd := localTime(savedEnd)

	if !restoredStart.Equal(start) || !restoredEnd.Equal(end) {
		t.Fatalf("изменился момент времени смены")
	}

	if restoredStart.Format("15:04") != "09:00" ||
		restoredEnd.Format("15:04") != "16:00" {
		t.Fatalf(
			"ожидалось 09:00–16:00, получено %s–%s",
			restoredStart.Format("15:04"),
			restoredEnd.Format("15:04"),
		)
	}
}

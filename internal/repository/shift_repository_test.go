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

	// В PostgreSQL хранится TIMESTAMP WITHOUT TIME ZONE.
	// Смена записана как 09:00–16:00 по времени Уфы.
	//
	// Имитируем чтение pgx, при котором время
	// получено с Location UTC, но часы остались прежними.
	savedStart := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	savedEnd := time.Date(2026, 11, 10, 16, 0, 0, 0, time.UTC)

	restoredStart := localTime(savedStart)
	restoredEnd := localTime(savedEnd)

	// Проверяем, что локальные часы не изменились.
	if restoredStart.Format("15:04") != "09:00" ||
		restoredEnd.Format("15:04") != "16:00" {
		t.Fatalf(
			"ожидалось 09:00–16:00, получено %s–%s",
			restoredStart.Format("15:04"),
			restoredEnd.Format("15:04"),
		)
	}

	// Проверяем, что восстановлен часовой пояс Уфы.
	expectedStart := time.Date(2026, 11, 10, 9, 0, 0, 0, ufa)
	expectedEnd := time.Date(2026, 11, 10, 16, 0, 0, 0, ufa)

	if !restoredStart.Equal(expectedStart) ||
		!restoredEnd.Equal(expectedEnd) {
		t.Fatal("неправильно восстановлен момент времени смены")
	}
}

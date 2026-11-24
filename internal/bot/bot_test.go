package bot

import (
	"testing"
	"time"
)

func TestStartOfShiftDayTimezone(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Yekaterinburg")
	if err != nil {
		t.Fatal(err)
	}

	date, err := time.ParseInLocation(
		"02.01.2006",
		"10.11.2026",
		loc,
	)
	if err != nil {
		t.Fatal(err)
	}

	day := startOfShiftDay(date)

	// Имитируем ввод начала смены в 09:00
	start := time.Date(
		day.Year(),
		day.Month(),
		day.Day(),
		9, 0, 0, 0,
		day.Location(),
	)

	// Проверяем момент времени, который получит PostgreSQL
	expectedUTC := "2026-11-10 04:00"
	actualUTC := start.UTC().Format("2006-01-02 15:04")

	if actualUTC != expectedUTC {
		t.Fatalf(
			"ожидалось UTC %s, получено %s",
			expectedUTC,
			actualUTC,
		)
	}
}

func TestIsActiveInlineMessage(t *testing.T) {
	b := &Bot{
		activeInlineMessages: map[int64]int{
			123: 456,
		},
	}

	tests := []struct {
		name      string
		chatID    int64
		messageID int
		want      bool
	}{
		{"Активное сообщение", 123, 456, true},
		{"Устаревшее сообщение", 123, 999, false},
		{"Неизвестный чат", 999, 456, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.isActiveInlineMessage(tt.chatID, tt.messageID)

			if got != tt.want {
				t.Errorf("ожидалось %v, получено %v", tt.want, got)
			}
		})
	}
}

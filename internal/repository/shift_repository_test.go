package repository

import (
	"testing"
	"time"
)

func TestLocalTime(t *testing.T) {
	startUTC := time.Date(2026, 11, 11, 9, 0, 0, 0, time.UTC)
	endUTC := time.Date(2026, 11, 11, 16, 0, 0, 0, time.UTC)

	start := localTime(startUTC)
	end := localTime(endUTC)

	if start.Format("15:04") != "14:00" ||
		end.Format("15:04") != "21:00" {
		t.Fatalf(
			"ожидалось 14:00–21:00, получено %s–%s",
			start.Format("15:04"),
			end.Format("15:04"),
		)
	}

	if !start.Equal(startUTC) || !end.Equal(endUTC) {
		t.Fatal("при преобразовании часового пояса изменился момент времени")
	}

	if start.Location().String() != "Asia/Yekaterinburg" {
		t.Fatalf("неверный часовой пояс: %s", start.Location())
	}
}

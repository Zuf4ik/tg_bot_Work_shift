package model

import "time"

type Shift struct {
	ID         int
	TelegramID int64
	StartTime  time.Time
	EndTime    time.Time
	Break      time.Duration
	HourlyRate float64
}

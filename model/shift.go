package model

import "time"

type Shift struct {
	ID         int
	StartTime  time.Time
	EndTime    time.Time
	Break      time.Duration
	HourlyRate float64
}

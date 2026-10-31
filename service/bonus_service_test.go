package service

import "testing"

func TestCalculateBonus(t *testing.T) {
	tests := []struct {
		name     string
		hours    float64
		norm     float64
		expected float64
	}{
		{"Меньше 80 часов", 79, 168, 0},
		{"80 часов", 80, 168, 1500},
		{"120 часов", 120, 168, 3000},
		{"140 часов", 140, 168, 8000},
		{"Выполнена норма", 168, 168, 15000},
		{"Норма меньше 140", 136, 136, 15000},
		{"Отрицательные часы", -5, 168, 0},
		{"Некорректная норма", 100, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateBonus(tt.hours, tt.norm)

			if got != tt.expected {
				t.Errorf(
					"ожидалось %.2f, получено %.2f",
					tt.expected,
					got,
				)
			}
		})
	}
}

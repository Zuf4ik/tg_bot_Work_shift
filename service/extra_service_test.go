package service

import (
	"math"
	"testing"
	"time"

	"tg_bot/internal/repository"
	"tg_bot/model"
)

func TestCalculateExtraSalary(t *testing.T) {
	// Смена 10:00–18:00, без перерыва.
	// Ставка 250 ₽/час.
	shift := model.Shift{
		StartTime:  time.Date(2026, 11, 3, 10, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 11, 3, 18, 0, 0, 0, time.Local),
		Break:      0,
		HourlyRate: 250,
	}

	tests := []struct {
		name     string
		extras   []repository.ExtraType
		expected float64
		wantErr  bool
	}{
		{
			name:     "Без дополнительных работ",
			expected: 0,
		},
		{
			name:     "Разгрузка",
			extras:   []repository.ExtraType{repository.ExtraUnloading},
			expected: 500,
		},
		{
			name:     "Мусор",
			extras:   []repository.ExtraType{repository.ExtraTrash},
			expected: 62.5,
		},
		{
			name:     "Доставка",
			extras:   []repository.ExtraType{repository.ExtraDelivery},
			expected: 1000,
		},
		{
			name: "Все дополнительные работы",
			extras: []repository.ExtraType{
				repository.ExtraUnloading,
				repository.ExtraTrash,
				repository.ExtraDelivery,
			},
			expected: 1562.5,
		},
		{
			name: "Повторная разгрузка",
			extras: []repository.ExtraType{
				repository.ExtraUnloading,
				repository.ExtraUnloading,
			},
			wantErr: true,
		},
		{
			name: "Неизвестная работа",
			extras: []repository.ExtraType{
				repository.ExtraType("unknown"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateExtraSalary(shift, tt.extras)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Ожидалась ошибка, но её нет")
				}
				return
			}

			if err != nil {
				t.Fatalf("Неожиданная ошибка: %v", err)
			}

			if math.Abs(got-tt.expected) > 0.001 {
				t.Errorf(
					"Получено %.2f ₽, ожидалось %.2f ₽",
					got,
					tt.expected,
				)
			}
		})
	}
}

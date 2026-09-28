package service

import (
	"testing"
	"time"

	"tg_bot/model"
)

func TestCalculateSalaryNormalShift(t *testing.T) {
	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 28, 10, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 28, 18, 0, 0, 0, time.Local),
		Break:      0,
		HourlyRate: 250,
	}

	salary, err := CalculateSalary(shift)

	if err != nil {
		t.Fatal(err)
	}

	expected := 2000.0

	if salary != expected {
		t.Errorf("ожидалось %.2f, получено %.2f", expected, salary)
	}
}

func TestCalculateSalaryNightShift(t *testing.T) {
	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 28, 22, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 29, 6, 0, 0, 0, time.Local),
		Break:      0,
		HourlyRate: 250,
	}

	salary, err := CalculateSalary(shift)

	if err != nil {
		t.Fatal(err)
	}

	// 8 часов × 250 × 1.3
	expected := 2600.0

	if salary != expected {
		t.Errorf("ожидалось %.2f, получено %.2f", expected, salary)
	}
}

func TestCalculateSalaryMixedShift(t *testing.T) {
	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 29, 2, 0, 0, 0, time.Local),
		Break:      0,
		HourlyRate: 250,
	}

	salary, err := CalculateSalary(shift)

	if err != nil {
		t.Fatal(err)
	}

	// 20:00–22:00:
	// 2 × 250 = 500
	//
	// 22:00–02:00:
	// 4 × 250 × 1.3 = 1300
	//
	// Итого: 1800
	expected := 1800.0

	if salary != expected {
		t.Errorf("ожидалось %.2f, получено %.2f", expected, salary)
	}
}

func TestCalculateSalaryWithBreak(t *testing.T) {
	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 28, 10, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local),
		Break:      1 * time.Hour,
		HourlyRate: 250,
	}

	salary, err := CalculateSalary(shift)

	if err != nil {
		t.Fatal(err)
	}

	// 10 часов смены - 1 час перерыва = 9 оплачиваемых часов.
	// 9 × 250 = 2250
	expected := 2250.0

	if salary != expected {
		t.Errorf("ожидалось %.2f, получено %.2f", expected, salary)
	}
}

func TestCalculateHours(t *testing.T) {
	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 28, 10, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local),
		Break:      1 * time.Hour,
		HourlyRate: 250,
	}

	hours, err := CalculateHours(shift)

	if err != nil {
		t.Fatal(err)
	}

	expected := 9.0

	if hours != expected {
		t.Errorf("ожидалось %.2f, получено %.2f", expected, hours)
	}
}

func TestCalculateHoursInvalidBreak(t *testing.T) {
	shift := model.Shift{
		StartTime: time.Date(2026, 9, 28, 10, 0, 0, 0, time.Local),
		EndTime:   time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local),
		Break:     11 * time.Hour,
	}

	_, err := CalculateHours(shift)

	if err == nil {
		t.Error("ожидалась ошибка из-за слишком большого перерыва")
	}
}

func TestCalculateSalaryInvalidRate(t *testing.T) {
	shift := model.Shift{
		StartTime:  time.Date(2026, 9, 28, 10, 0, 0, 0, time.Local),
		EndTime:    time.Date(2026, 9, 28, 18, 0, 0, 0, time.Local),
		Break:      0,
		HourlyRate: -250,
	}

	_, err := CalculateSalary(shift)

	if err == nil {
		t.Error("ожидалась ошибка из-за отрицательной ставки")
	}
}

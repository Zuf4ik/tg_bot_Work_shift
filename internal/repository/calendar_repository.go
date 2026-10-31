package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CalendarRepository struct {
	db *pgxpool.Pool
}

func NewCalendarRepository(db *pgxpool.Pool) *CalendarRepository {
	return &CalendarRepository{db: db}
}

func (r *CalendarRepository) GetMonthlyNorm(
	ctx context.Context,
	year int,
	month int,
) (float64, error) {
	if year < 2020 || year > 2036 {
		return 0, fmt.Errorf("некорректный год: %d", year)
	}

	if month < 1 || month > 12 {
		return 0, fmt.Errorf("некорректный месяц: %d", month)
	}

	var norm float64

	query := `
		SELECT norm_hours
		FROM production_calendar
		WHERE year = $1 AND month = $2
	`

	err := r.db.QueryRow(ctx, query, year, month).Scan(&norm)
	if err != nil {
		return 0, fmt.Errorf(
			"не удалось получить норму за %02d.%d: %w",
			month, year, err,
		)
	}

	return norm, nil
}

package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) SetHourlyRate(
	ctx context.Context,
	telegramID int64,
	hourlyRate float64,
) error {
	query := `
		INSERT INTO users (telegram_id, hourly_rate)
		VALUES ($1, $2)
		ON CONFLICT (telegram_id)
		DO UPDATE SET hourly_rate = EXCLUDED.hourly_rate
	`

	_, err := r.db.Exec(
		ctx,
		query,
		telegramID,
		hourlyRate,
	)

	return err
}

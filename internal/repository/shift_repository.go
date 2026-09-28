package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tg_bot/model"
)

type ShiftRepository struct {
	db *pgxpool.Pool
}

func NewShiftRepository(db *pgxpool.Pool) *ShiftRepository {
	return &ShiftRepository{
		db: db,
	}
}

func (r *ShiftRepository) CreateShift(ctx context.Context, shift model.Shift) (int, error) {
	var id int

	query := `
		INSERT INTO shifts (
			start_time,
			end_time,
			break_minutes,
			hourly_rate
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.db.QueryRow(
		ctx,
		query,
		shift.StartTime,
		shift.EndTime,
		int(shift.Break.Minutes()),
		shift.HourlyRate,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *ShiftRepository) GetShifts(ctx context.Context) ([]model.Shift, error) {
	query := `
		SELECT
			id,
			start_time,
			end_time,
			break_minutes,
			hourly_rate
		FROM shifts
		ORDER BY start_time DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shifts []model.Shift

	for rows.Next() {
		var shift model.Shift
		var breakMinutes int

		err := rows.Scan(
			&shift.ID,
			&shift.StartTime,
			&shift.EndTime,
			&breakMinutes,
			&shift.HourlyRate,
		)
		if err != nil {
			return nil, err
		}

		shift.Break = time.Duration(breakMinutes) * time.Minute

		shifts = append(shifts, shift)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return shifts, nil
}

func (r *ShiftRepository) DeleteShift(ctx context.Context, id int) error {
	query := `
		DELETE FROM shifts
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, id)

	return err
}

func (r *ShiftRepository) UpdateShift(ctx context.Context, id int, shift model.Shift) error {
	query := `
		UPDATE shifts
		SET
			start_time = $1,
			end_time = $2,
			break_minutes = $3,
			hourly_rate = $4
		WHERE id = $5
	`

	_, err := r.db.Exec(
		ctx,
		query,
		shift.StartTime,
		shift.EndTime,
		int(shift.Break.Minutes()),
		shift.HourlyRate,
		id,
	)

	return err
}

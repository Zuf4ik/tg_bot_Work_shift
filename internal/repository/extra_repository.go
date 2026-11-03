package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExtraType string

const (
	ExtraUnloading ExtraType = "unloading"
	ExtraTrash     ExtraType = "trash"
	ExtraDelivery  ExtraType = "delivery"
)

type ExtraRepository struct {
	db *pgxpool.Pool
}

func NewExtraRepository(db *pgxpool.Pool) *ExtraRepository {
	return &ExtraRepository{db: db}
}

// AddExtra добавляет дополнительную работу к смене.
// telegramID нужен, чтобы пользователь не мог изменить чужую смену.
func (r *ExtraRepository) AddExtra(
	ctx context.Context,
	telegramID int64,
	shiftID int,
	extraType ExtraType,
) error {
	if shiftID <= 0 {
		return fmt.Errorf("некорректный ID смены")
	}

	if !validExtraType(extraType) {
		return fmt.Errorf("неизвестный тип дополнительной работы")
	}

	tag, err := r.db.Exec(ctx, `
        INSERT INTO shift_extras (shift_id, extra_type)
        SELECT id, $3
        FROM shifts
        WHERE id = $1 AND telegram_id = $2
        ON CONFLICT (shift_id, extra_type) DO NOTHING
    `, shiftID, telegramID, extraType)

	if err != nil {
		return fmt.Errorf("ошибка добавления работы: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("смена не найдена или работа уже добавлена")
	}

	return nil
}

// GetExtras возвращает дополнительные работы конкретной смены.
func (r *ExtraRepository) GetExtras(
	ctx context.Context,
	telegramID int64,
	shiftID int,
) ([]ExtraType, error) {
	rows, err := r.db.Query(ctx, `
        SELECT se.extra_type
        FROM shift_extras se
        JOIN shifts s ON s.id = se.shift_id
        WHERE se.shift_id = $1 AND s.telegram_id = $2
        ORDER BY se.id
    `, shiftID, telegramID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var extras []ExtraType

	for rows.Next() {
		var extra ExtraType
		if err := rows.Scan(&extra); err != nil {
			return nil, err
		}
		extras = append(extras, extra)
	}

	return extras, rows.Err()
}

// DeleteExtra удаляет дополнительную работу со смены.
func (r *ExtraRepository) DeleteExtra(
	ctx context.Context,
	telegramID int64,
	shiftID int,
	extraType ExtraType,
) error {
	if !validExtraType(extraType) {
		return fmt.Errorf("неизвестный тип дополнительной работы")
	}

	tag, err := r.db.Exec(ctx, `
        DELETE FROM shift_extras se
        USING shifts s
        WHERE se.shift_id = s.id
          AND se.shift_id = $1
          AND s.telegram_id = $2
          AND se.extra_type = $3
    `, shiftID, telegramID, extraType)

	if err != nil {
		return fmt.Errorf("ошибка удаления работы: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("дополнительная работа не найдена")
	}

	return nil
}

func validExtraType(extraType ExtraType) bool {
	switch extraType {
	case ExtraUnloading, ExtraTrash, ExtraDelivery:
		return true
	default:
		return false
	}
}

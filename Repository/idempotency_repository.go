package repository

import (
	"context"
	"database/sql"
)

type IdempotencyRepository struct {
	db *sql.DB
}

func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{
		db: db,
	}
}

func (r *IdempotencyRepository) GetReferenceNumber(
	ctx context.Context,
	tx *sql.Tx,
	idempotencyKey string,
) (string, error) {
	query := `SELECT reference_number FROM idempotency_keys WHERE idempotency_key = ?`
	var refNumber string
	// Use QueryRowContext instead of QueryRow
	err := tx.QueryRowContext(ctx, query, idempotencyKey).Scan(&refNumber)
	if err != nil {
		return "", err
	}
	return refNumber, nil
}

func (r *IdempotencyRepository) Create(
	ctx context.Context,
	tx *sql.Tx,
	idempotencyKey string,
	referenceNumber string,
) error {
	query := `
		INSERT INTO idempotency_keys (idempotency_key, reference_number, created_at)
		VALUES (?, ?, NOW())
	`
	// Use ExecContext instead of Exec
	_, err := tx.ExecContext(ctx, query, idempotencyKey, referenceNumber)
	return err
}

package repository

import (
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
	tx *sql.Tx,
	idempotencyKey string,
) (string, error) {
	var referenceNumber string

	err := tx.QueryRow(
		`SELECT reference_number
			FROM idempotency_keys
			WHERE idempotency_key = ?`, idempotencyKey,
	).Scan(
		&referenceNumber,
	)

	if err != nil {
		return "", err
	}

	return referenceNumber, nil

}

func (r *IdempotencyRepository) Create(
	tx *sql.Tx,
	idempotencyKey string,
	referenceNumber string,
) error {

	_, err := tx.Exec(
		`INSERT INTO idempotency_keys
		(
			idempotency_key,
			 reference_number
		) VALUES (?,?)`,
		idempotencyKey,
		referenceNumber,
	)

	return err
}

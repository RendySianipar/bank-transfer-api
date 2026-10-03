package repository

import (
	"bank-transfer-api/model"
	"context"
	"database/sql"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{
		db: db,
	}
}

func (r *AccountRepository) GetAccountForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	accountID string,
) (*model.Account, error) {
	query := `
		SELECT id, owner_name, balance, status
		FROM accounts
		WHERE id = ?
		FOR UPDATE
	`

	var account model.Account
	// Use QueryRowContext instead of QueryRow
	err := tx.QueryRowContext(ctx, query, accountID).Scan(
		&account.ID,
		&account.OwnerName,
		&account.Balance,
		&account.Status,
	)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (r *AccountRepository) GetAccountForUpdateByUser(
	ctx context.Context,
	tx *sql.Tx,
	accountID string,
	userID string,
) (*model.Account, error) {
	query := `
		SELECT id, owner_name, balance, status, user_id
		FROM accounts
		WHERE id = ? AND user_id = ?
		FOR UPDATE
	`

	var account model.Account
	// Use QueryRowContext instead of QueryRow
	err := tx.QueryRowContext(ctx, query, accountID, userID).Scan(
		&account.ID,
		&account.OwnerName,
		&account.Balance,
		&account.Status,
		&account.UserID,
	)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (r *AccountRepository) DeductBalance(
	ctx context.Context,
	tx *sql.Tx,
	accountID string,
	amount int64,
) error {
	query := `UPDATE accounts SET balance = balance - ? WHERE id = ?`
	// Use ExecContext instead of Exec
	_, err := tx.ExecContext(ctx, query, amount, accountID)
	return err
}

func (r *AccountRepository) AddBalance(
	ctx context.Context,
	tx *sql.Tx,
	accountID string,
	amount int64,
) error {
	query := `UPDATE accounts SET balance = balance + ? WHERE id = ?`
	// Use ExecContext instead of Exec
	_, err := tx.ExecContext(ctx, query, amount, accountID)
	return err
}

func (r *AccountRepository) CreateTransfer(
	ctx context.Context,
	tx *sql.Tx,
	transfer model.Transfer,
) error {
	query := `
		INSERT INTO transfers (id, reference_number, from_account_id, to_account_id, amount, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	// Use ExecContext instead of Exec
	_, err := tx.ExecContext(
		ctx,
		query,
		transfer.ID,
		transfer.ReferenceNumber,
		transfer.FromAccountID,
		transfer.ToAccountID,
		transfer.Amount,
		transfer.Status,
		transfer.CreatedAt,
	)
	return err
}

func (r *AccountRepository) GetAllTransfer() ([]model.Transfer, error) {
	query := `
		SELECT id, reference_number, from_account_id, to_account_id, amount, status, created_at
		FROM transfers
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transfers []model.Transfer
	for rows.Next() {
		var t model.Transfer
		err := rows.Scan(
			&t.ID,
			&t.ReferenceNumber,
			&t.FromAccountID,
			&t.ToAccountID,
			&t.Amount,
			&t.Status,
			&t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, t)
	}

	return transfers, rows.Err()
}

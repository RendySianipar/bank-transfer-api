package repository

import (
	"bank-transfer-api/model"
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
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
	tx *sql.Tx,
	accountID string,
) (*model.Account, error) {

	var account model.Account

	err := tx.QueryRow(
		`SELECT id, owner_name, balance, status
		FROM accounts
		WHERE id = ?
		FOR UPDATE`,
		accountID,
	).Scan(
		&accountID,
		&account.OwnerName,
		&account.Balance,
		&account.Status,
	)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (r *AccountRepository) DeductBalance(
	tx *sql.Tx,
	accountID string,
	amount int64,
) error {

	_, err := tx.Exec(
		`UPDATE accounts
		SET balance = balance - ?
		WHERE id = ?`,
		amount,
		accountID,
	)

	return err
}

func (r *AccountRepository) AddBalance(
	tx *sql.Tx,
	accountID string,
	amount int64,
) error {

	_, err := tx.Exec(
		`UPDATE accounts
		SET balance = balance + ?
		WHERE id = ?`,
		amount,
		accountID,
	)

	return err
}

func (r *AccountRepository) CreateTransfer(
	tx *sql.Tx,
	transfer model.Transfer,
) error {
	_, err := tx.Exec(
		`INSERT INTO transfers 
		(
		id, reference_number, from_account_id, to_account_id, amount, status, created_at
		)
		VALUES (?,?,?,?,?,?,?)`,
		transfer.ID, transfer.ReferenceNumber, transfer.FromAccountID, transfer.ToAccountID, transfer.Amount, transfer.Status, transfer.CreatedAt,
	)

	return err
}

func (r *AccountRepository) GetAllTransfer() ([]model.Transfer, error) {

	rows, err := r.db.Query(
		`SELECT id, reference_number, from_account_id, to_account_id, amount, status, created_at
		 FROM transfers`,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var allTrf []model.Transfer

	for rows.Next() {
		var trf model.Transfer

		if err := rows.Scan(
			&trf.ID,
			&trf.ReferenceNumber,
			&trf.FromAccountID,
			&trf.ToAccountID,
			&trf.Amount,
			&trf.Status,
			&trf.CreatedAt,
		); err != nil {
			return nil, err
		}
		allTrf = append(allTrf, trf)
	}

	// rows.Next() also stops on a scan/network error mid-iteration -
	// rows.Err() surfaces that, since the loop above would otherwise swallow it silently.
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return allTrf, nil
}

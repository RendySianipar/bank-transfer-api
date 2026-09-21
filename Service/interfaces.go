package service

import (
	"bank-transfer-api/model"
	"database/sql"
)

// AccountRepositoryInterface describes exactly what TransferService needs
// from an account repository - nothing more.
//
// The REAL *repository.AccountRepository already has all these methods,
// so it automatically satisfies this interface - no changes needed there.
// In unit tests, we provide a FAKE implementation instead (no real DB).
type AccountRepositoryInterface interface {
	GetAccountForUpdate(tx *sql.Tx, accountID string) (*model.Account, error)
	GetAccountForUpdateByUser(tx *sql.Tx, accountID string, userID string) (*model.Account, error)
	DeductBalance(tx *sql.Tx, accountID string, amount int64) error
	AddBalance(tx *sql.Tx, accountID string, amount int64) error
	CreateTransfer(tx *sql.Tx, transfer model.Transfer) error
	GetAllTransfer() ([]model.Transfer, error)
}

// IdempotencyRepositoryInterface describes exactly what TransferService
// needs from an idempotency repository.
type IdempotencyRepositoryInterface interface {
	GetReferenceNumber(tx *sql.Tx, idempotencyKey string) (string, error)
	Create(tx *sql.Tx, idempotencyKey string, referenceNumber string) error
}

type UserRepositoryInterface interface {
	CreateUser(user model.User) error
	GetUserByUsername(username string) (*model.User, error)
}

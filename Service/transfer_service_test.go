package service

import (
	"bank-transfer-api/model"
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// ---------------------------------------------------------------------
// Fake repositories. These satisfy AccountRepositoryInterface and
// IdempotencyRepositoryInterface, but hold data in plain Go maps
// instead of talking to a real database. They ignore the *sql.Tx
// argument entirely - which is fine, because in these tests the
// *sql.Tx only needs to be "a real transaction object that Commit()/
// Rollback() can be called on", not something our fakes read from.
// ---------------------------------------------------------------------

type fakeAccountRepository struct {
	accounts  map[string]*model.Account // keyed by account ID
	transfers []model.Transfer
}

func newFakeAccountRepository() *fakeAccountRepository {
	return &fakeAccountRepository{
		accounts: make(map[string]*model.Account),
	}
}

func (f *fakeAccountRepository) GetAccountForUpdate(tx *sql.Tx, accountID string) (*model.Account, error) {
	acc, ok := f.accounts[accountID]
	if !ok {
		return nil, sql.ErrNoRows
	}
	accCopy := *acc // return a COPY so test code can't mutate fake state by accident
	return &accCopy, nil
}

func (f *fakeAccountRepository) GetAccountForUpdateByUser(tx *sql.Tx, accountID string, userID string) (*model.Account, error) {
	acc, ok := f.accounts[accountID]
	if !ok || acc.UserID != userID {
		return nil, sql.ErrNoRows
	}
	accCopy := *acc
	return &accCopy, nil
}

func (f *fakeAccountRepository) DeductBalance(tx *sql.Tx, accountID string, amount int64) error {
	acc, ok := f.accounts[accountID]
	if !ok {
		return sql.ErrNoRows
	}
	acc.Balance -= amount
	return nil
}

func (f *fakeAccountRepository) AddBalance(tx *sql.Tx, accountID string, amount int64) error {
	acc, ok := f.accounts[accountID]
	if !ok {
		return sql.ErrNoRows
	}
	acc.Balance += amount
	return nil
}

func (f *fakeAccountRepository) CreateTransfer(tx *sql.Tx, transfer model.Transfer) error {
	f.transfers = append(f.transfers, transfer)
	return nil
}

func (f *fakeAccountRepository) GetAllTransfer() ([]model.Transfer, error) {
	return f.transfers, nil
}

type fakeIdempotencyRepository struct {
	keys map[string]string // idempotencyKey -> referenceNumber
}

func newFakeIdempotencyRepository() *fakeIdempotencyRepository {
	return &fakeIdempotencyRepository{keys: make(map[string]string)}
}

func (f *fakeIdempotencyRepository) GetReferenceNumber(tx *sql.Tx, idempotencyKey string) (string, error) {
	ref, ok := f.keys[idempotencyKey]
	if !ok {
		return "", sql.ErrNoRows
	}
	return ref, nil
}

func (f *fakeIdempotencyRepository) Create(tx *sql.Tx, idempotencyKey string, referenceNumber string) error {
	f.keys[idempotencyKey] = referenceNumber
	return nil
}

// ---------------------------------------------------------------------
// Test helper. Builds a TransferService wired to:
//   - a REAL *sql.DB, backed by sqlmock (so Begin/Commit/Rollback work
//     without panicking and without a real MySQL connection)
//   - FAKE repositories (so the actual data logic is in-memory and fast)
// ---------------------------------------------------------------------

func newTestService(t *testing.T) (svc *TransferService, accountRepo *fakeAccountRepository, mock sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// NOTE: we do NOT call mock.ExpectBegin() here. Transfer() validates
	// input BEFORE calling tx.Begin() (see the validation block at the
	// top of the function) - so tests that only exercise validation
	// errors never reach Begin() at all. Each test below registers
	// exactly the expectations that its specific code path will trigger.

	accountRepo = newFakeAccountRepository()
	accountRepo.accounts["ACC001"] = &model.Account{
		ID: "ACC001", OwnerName: "Ethan", Balance: 1000000, Status: "active", UserID: "user-1",
	}
	accountRepo.accounts["ACC002"] = &model.Account{
		ID: "ACC002", OwnerName: "Wick", Balance: 500000, Status: "active", UserID: "user-2",
	}

	idempotencyRepo := newFakeIdempotencyRepository()

	svc = NewTransferService(db, accountRepo, idempotencyRepo)

	return svc, accountRepo, mock
}

// ---------------------------------------------------------------------
// Test cases - table-driven style for the validation checks (the
// idiomatic Go pattern for testing many input/output combinations of
// the same function), separate focused tests for the more involved
// scenarios (insufficient balance, ownership, success, idempotency).
// ---------------------------------------------------------------------

func TestTransfer_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     model.TransferRequest
		userID  string
		wantErr string
	}{
		{
			name:    "empty from account",
			req:     model.TransferRequest{FromAccountID: "", ToAccountID: "ACC002", Amount: 1000},
			userID:  "user-1",
			wantErr: "from account ID is required",
		},
		{
			name:    "empty to account",
			req:     model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "", Amount: 1000},
			userID:  "user-1",
			wantErr: "to account ID is required",
		},
		{
			name:    "same sender and recipient",
			req:     model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC001", Amount: 1000},
			userID:  "user-1",
			wantErr: "sender and recepient cannot be the same",
		},
		{
			name:    "zero amount",
			req:     model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 0},
			userID:  "user-1",
			wantErr: "amount must be greater than 0",
		},
		{
			name:    "negative amount",
			req:     model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: -500},
			userID:  "user-1",
			wantErr: "amount must be greater than 0",
		},
		{
			name:    "amount exceeds max limit",
			req:     model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 10000001},
			userID:  "user-1",
			wantErr: "amount exceeds maximum transfer limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, mock := newTestService(t)
			// No ExpectBegin()/ExpectRollback() here - all these cases
			// fail validation BEFORE tx.Begin() is ever reached.

			_, err := svc.Transfer(tt.req, tt.userID, "idem-"+tt.name)

			if err == nil {
				t.Fatalf("expected error %q, got nil", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Errorf("expected error %q, got %q", tt.wantErr, err.Error())
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet sqlmock expectations: %v", err)
			}
		})
	}
}

func TestTransfer_InsufficientBalance(t *testing.T) {
	svc, _, mock := newTestService(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	req := model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 2000000} // more than the seeded 1,000,000

	_, err := svc.Transfer(req, "user-1", "idem-insufficient")

	if err == nil {
		t.Fatal("expected insufficient balance error, got nil")
	}
	if err.Error() != "insufficient balance" {
		t.Errorf("expected 'insufficient balance', got %q", err.Error())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestTransfer_WrongOwner(t *testing.T) {
	svc, _, mock := newTestService(t)
	mock.ExpectBegin()
	mock.ExpectRollback()

	// ACC001 belongs to user-1, not user-2 - this exercises the
	// authorization check (GetAccountForUpdateByUser).
	req := model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 1000}

	_, err := svc.Transfer(req, "user-2", "idem-wrong-owner")

	if err == nil {
		t.Fatal("expected sender account not found error, got nil")
	}
	if err.Error() != "sender account not found" {
		t.Errorf("expected 'sender account not found', got %q", err.Error())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestTransfer_Success(t *testing.T) {
	svc, accountRepo, mock := newTestService(t)
	mock.ExpectBegin()
	mock.ExpectCommit()

	req := model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 200000}

	refNumber, err := svc.Transfer(req, "user-1", "idem-success-1")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if refNumber == "" {
		t.Error("expected a non-empty reference number")
	}

	fromAcc := accountRepo.accounts["ACC001"]
	toAcc := accountRepo.accounts["ACC002"]

	if fromAcc.Balance != 800000 {
		t.Errorf("expected ACC001 balance 800000, got %d", fromAcc.Balance)
	}
	if toAcc.Balance != 700000 {
		t.Errorf("expected ACC002 balance 700000, got %d", toAcc.Balance)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestTransfer_IdempotentReplay(t *testing.T) {
	svc, accountRepo, mock := newTestService(t)

	req := model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 200000}
	idemKey := "idem-replay-same-key"

	// First call: goes all the way through, commits.
	mock.ExpectBegin()
	mock.ExpectCommit()
	firstRef, err := svc.Transfer(req, "user-1", idemKey)
	if err != nil {
		t.Fatalf("first call: expected no error, got %v", err)
	}

	// Second call: SAME idempotency key. Transfer() finds the existing
	// reference number and returns early - before ever reaching
	// Commit(), so the earlier `defer tx.Rollback()` fires instead
	// (harmlessly, since nothing was written in this branch).
	mock.ExpectBegin()
	mock.ExpectRollback()
	secondRef, err := svc.Transfer(req, "user-1", idemKey)
	if err != nil {
		t.Fatalf("second call: expected no error, got %v", err)
	}

	if firstRef != secondRef {
		t.Errorf("expected same reference number on replay, got %q vs %q", firstRef, secondRef)
	}

	// Critically: balance should reflect ONLY ONE transfer, not two.
	fromAcc := accountRepo.accounts["ACC001"]
	if fromAcc.Balance != 800000 {
		t.Errorf("expected balance to change only once (800000), got %d - money may have moved twice!", fromAcc.Balance)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// Sanity check that our fake correctly reports "not found" via
// errors.Is against sql.ErrNoRows, matching how the real repository
// behaves - important because Transfer() specifically checks
// `err == sql.ErrNoRows` to turn it into "sender account not found".
func TestFakeAccountRepository_NotFound(t *testing.T) {
	repo := newFakeAccountRepository()

	_, err := repo.GetAccountForUpdate(nil, "DOES_NOT_EXIST")

	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

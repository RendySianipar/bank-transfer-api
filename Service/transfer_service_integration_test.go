//go:build integration

package service

import (
	repository "bank-transfer-api/Repository"
	"bank-transfer-api/database/testdb"
	"bank-transfer-api/model"
	"database/sql"
	"sync"
	"testing"
)

// newIntegrationTestService wires a TransferService to REAL repositories
// backed by a REAL MySQL connection (the integration test database) -
// unlike Stage 11's newTestService, nothing here is faked. This is what
// actually exercises your real SQL strings, including FOR UPDATE.
func newIntegrationTestService(t *testing.T) (*TransferService, *sql.DB) {
	t.Helper()

	db := testdb.Connect(t)
	testdb.Reset(t, db) // start every test from a clean slate

	accountRepo := repository.NewAccountRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)

	svc := NewTransferService(db, accountRepo, idempotencyRepo)

	return svc, db
}

// seedAccounts inserts a minimal set of users + accounts directly via
// SQL (bypassing AuthService/bcrypt entirely - these tests care about
// TRANSFER correctness, not password hashing, so a real bcrypt hash
// would only slow tests down for no benefit here).
func seedAccounts(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(
		`INSERT INTO users (id, username, password_hash) VALUES
		('user-1', 'ethan', 'unused-hash'),
		('user-2', 'wick', 'unused-hash')`,
	)
	if err != nil {
		t.Fatalf("failed to seed users: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO accounts (id, owner_name, balance, status, user_id) VALUES
		('ACC001', 'Ethan', 1000000, 'active', 'user-1'),
		('ACC002', 'Wick', 500000, 'active', 'user-2')`,
	)
	if err != nil {
		t.Fatalf("failed to seed accounts: %v", err)
	}
}

func TestTransferIntegration_Success(t *testing.T) {
	svc, db := newIntegrationTestService(t)
	seedAccounts(t, db)

	req := model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 200000}

	refNumber, err := svc.Transfer(req, "user-1", "integration-idem-success")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if refNumber == "" {
		t.Error("expected a non-empty reference number")
	}

	// Verify against the REAL database - this is what actually proves
	// the SQL in DeductBalance/AddBalance is correct, not just that
	// our earlier fake's in-memory map math worked.
	var fromBalance, toBalance int64
	if err := db.QueryRow("SELECT balance FROM accounts WHERE id = 'ACC001'").Scan(&fromBalance); err != nil {
		t.Fatalf("failed to query ACC001 balance: %v", err)
	}
	if err := db.QueryRow("SELECT balance FROM accounts WHERE id = 'ACC002'").Scan(&toBalance); err != nil {
		t.Fatalf("failed to query ACC002 balance: %v", err)
	}

	if fromBalance != 800000 {
		t.Errorf("expected ACC001 balance 800000, got %d", fromBalance)
	}
	if toBalance != 700000 {
		t.Errorf("expected ACC002 balance 700000, got %d", toBalance)
	}

	// Also verify a transfer row was actually persisted.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM transfers WHERE reference_number = ?", refNumber).Scan(&count); err != nil {
		t.Fatalf("failed to query transfers: %v", err)
	}
	if count != 1 {
		t.Errorf("expected exactly 1 transfer row with this reference number, got %d", count)
	}
}

func TestTransferIntegration_WrongOwner(t *testing.T) {
	svc, db := newIntegrationTestService(t)
	seedAccounts(t, db)

	// ACC001 belongs to user-1 (per seedAccounts), not user-2.
	// This exercises the REAL "WHERE id = ? AND user_id = ?" SQL,
	// not a fake's in-memory ownership check.
	req := model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: 1000}

	_, err := svc.Transfer(req, "user-2", "integration-idem-wrong-owner")

	if err == nil {
		t.Fatal("expected sender account not found error, got nil")
	}
	if err.Error() != "sender account not found" {
		t.Errorf("expected 'sender account not found', got %q", err.Error())
	}

	// Confirm no balance changed at all - the transaction should have
	// rolled back cleanly.
	var balance int64
	if err := db.QueryRow("SELECT balance FROM accounts WHERE id = 'ACC001'").Scan(&balance); err != nil {
		t.Fatalf("failed to query ACC001 balance: %v", err)
	}
	if balance != 1000000 {
		t.Errorf("expected balance unchanged at 1000000, got %d", balance)
	}
}

// TestTransferIntegration_ConcurrentTransfers formalizes what Stage 10's
// cmd/concurrency_test/main.go verified manually (by eyeballing MySQL
// Workbench output) into an actual, automatically-asserted go test.
//
// This is the single most important thing an integration test can prove
// that a unit test with fakes never can: does "FOR UPDATE" really
// serialize concurrent access against a REAL database engine? A fake
// repository's in-memory map has no locking at all, so Stage 11's unit
// tests were structurally incapable of catching a broken/missing
// FOR UPDATE - only a real database can expose that bug.
func TestTransferIntegration_ConcurrentTransfers(t *testing.T) {
	svc, db := newIntegrationTestService(t)
	seedAccounts(t, db)
	// ACC001 seeded with balance 1,000,000 (see seedAccounts).

	const numRequests = 10
	const transferAmount = 200000 // 10 x 200,000 = 2,000,000 requested, but only 1,000,000 available -> exactly 5 should succeed

	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex // protects successCount from concurrent writes

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			req := model.TransferRequest{FromAccountID: "ACC001", ToAccountID: "ACC002", Amount: transferAmount}

			// Each goroutine needs its own idempotency key - otherwise
			// they'd all be treated as retries of the SAME request,
			// which would defeat the point of this test (we want 10
			// genuinely separate transfer attempts racing each other).
			idemKey := "integration-concurrent-" + string(rune('a'+i))

			_, err := svc.Transfer(req, "user-1", idemKey)

			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	if successCount != 5 {
		t.Errorf("expected exactly 5 successful transfers (5 x 200000 = 1000000 available), got %d", successCount)
	}

	var finalBalance int64
	if err := db.QueryRow("SELECT balance FROM accounts WHERE id = 'ACC001'").Scan(&finalBalance); err != nil {
		t.Fatalf("failed to query final balance: %v", err)
	}

	if finalBalance != 0 {
		t.Errorf("expected final balance exactly 0, got %d - if negative, FOR UPDATE failed to prevent overdraft", finalBalance)
	}
}

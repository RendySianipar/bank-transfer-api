package service

import (
	repository "bank-transfer-api/Repository"
	"bank-transfer-api/model"
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

type TransferService struct {
	db                    *sql.DB
	accountRepository     *repository.AccountRepository
	idempotencyRepository *repository.IdempotencyRepository
}

func NewTransferService(
	db *sql.DB,
	accountRepository *repository.AccountRepository,
	idempotencyRepository *repository.IdempotencyRepository,
) *TransferService {
	return &TransferService{
		db:                    db,
		accountRepository:     accountRepository,
		idempotencyRepository: idempotencyRepository,
	}
}

func (s *TransferService) GetAllTransfer() ([]model.Transfer, error) {

	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	allTrf, err := s.accountRepository.GetAllTransfer(tx)

	return allTrf, err
}

func (s *TransferService) Transfer(
	req model.TransferRequest,
	idempotencyKey string,
) (string, error) {

	// 1. Basic validation
	if req.Amount <= 0 {
		return "", errors.New("amount must be greater than 0")
	}

	if req.FromAccountID == req.ToAccountID {
		return "", errors.New("sender and recipient cannot be the same")
	}

	// 2. Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}

	// Safety net:
	// If Commit() is not called, transaction will be rolled back.
	defer tx.Rollback()

	referenceNumber, err := s.idempotencyRepository.GetReferenceNumber(
		tx,
		idempotencyKey,
	)

	if err == nil {
		return referenceNumber, err
	}

	if err != sql.ErrNoRows {
		return "", err
	}

	// Get sender
	fromAccount, err := s.accountRepository.GetAccountForUpdate(
		tx,
		req.FromAccountID,
	)

	if err == sql.ErrNoRows {
		return "", errors.New("sender account not found")
	}

	if err != nil {
		return "", err
	}

	if fromAccount.Status != "active" {
		return "", errors.New("sender account is not active")
	}

	// Get recipient
	toAccount, err := s.accountRepository.GetAccountForUpdate(
		tx,
		req.ToAccountID,
	)

	if err == sql.ErrNoRows {
		return "", errors.New("recipient account not found")
	}

	if err != nil {
		return "", err
	}

	if toAccount.Status != "active" {
		return "", errors.New("recipient account is not active")
	}

	// Deduct sender
	err = s.accountRepository.DeductBalance(
		tx,
		req.FromAccountID,
		req.Amount,
	)

	if err != nil {
		return "", err
	}

	// Add recipient
	err = s.accountRepository.AddBalance(
		tx,
		req.ToAccountID,
		req.Amount,
	)

	if err != nil {
		return "", err
	}

	if err != nil {
		return "", err
	}

	referenceNumber, err = GenerateReferenceNumber(12)
	if err != nil {
		return "", err
	}

	transfer := model.Transfer{
		ID:              uuid.New().String(),
		ReferenceNumber: referenceNumber,
		FromAccountID:   req.FromAccountID,
		ToAccountID:     req.ToAccountID,
		Amount:          req.Amount,
		Status:          "success",
		CreatedAt:       time.Now(),
	}

	err = s.accountRepository.CreateTransfer(
		tx,
		transfer,
	)

	if err != nil {
		return "", err
	}

	err = s.idempotencyRepository.Create(
		tx,
		idempotencyKey,
		referenceNumber,
	)

	if err != nil {
		return "", err
	}

	// Commit
	if err := tx.Commit(); err != nil {
		return "", err
	}

	return transfer.ReferenceNumber, nil
}

// func GenerateReferenceNumber() string {
// 	now := time.Now()

// 	randomNumber := rand.Intn(1000000)

// 	return fmt.Sprintf("TRX-%s-%06d", now.Format("20060102"), randomNumber)
// }

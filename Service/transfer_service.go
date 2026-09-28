package service

import (
	"bank-transfer-api/model"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Create the tracer at package level (once, not in every function)
var tracer = otel.Tracer("bank-transfer-api/transfer")

type TransferService struct {
	db                    *sql.DB
	accountRepository     AccountRepositoryInterface
	idempotencyRepository IdempotencyRepositoryInterface
}

func NewTransferService(
	db *sql.DB,
	accountRepository AccountRepositoryInterface,
	idempotencyRepository IdempotencyRepositoryInterface,
) *TransferService {
	return &TransferService{
		db:                    db,
		accountRepository:     accountRepository,
		idempotencyRepository: idempotencyRepository,
	}
}

func (s *TransferService) GetAllTransfer() ([]model.Transfer, error) {
	allTrf, err := s.accountRepository.GetAllTransfer()
	if err != nil {
		return nil, err
	}

	return allTrf, err
}

// Updated: now accepts ctx as first parameter
func (s *TransferService) Transfer(
	ctx context.Context,
	req model.TransferRequest,
	userID string,
	idempotencyKey string,
) (string, error) {
	// Create a span for this transfer operation
	ctx, span := tracer.Start(ctx, "transfer")
	defer span.End()

	// Set span attributes (what data is being transferred)
	span.SetAttributes(
		attribute.String("transfer.from_account", req.FromAccountID),
		attribute.String("transfer.to_account", req.ToAccountID),
		attribute.Int64("transfer.amount", req.Amount),
	)

	// Validate account IDs
	if req.FromAccountID == "" {
		err := errors.New("from account ID is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		return "", err
	}

	if req.ToAccountID == "" {
		err := errors.New("to account ID is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		return "", err
	}

	if req.FromAccountID == req.ToAccountID {
		err := errors.New("sender and recepient cannot be the same")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		return "", err
	}

	// Validate amount
	if req.Amount <= 0 {
		err := errors.New("amount must be greater than 0")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		return "", err
	}

	if req.Amount > 10000000 {
		err := errors.New("amount exceeds maximum transfer limit")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		return "", err
	}

	// Start transaction
	tx, err := s.db.Begin()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction begin failed")
		return "", err
	}

	defer tx.Rollback()

	referenceNumber, err := s.idempotencyRepository.GetReferenceNumber(
		tx,
		idempotencyKey,
	)

	if err == nil {
		span.SetStatus(codes.Ok, "idempotency hit")
		return referenceNumber, err
	}

	if err != sql.ErrNoRows {
		span.RecordError(err)
		span.SetStatus(codes.Error, "idempotency lookup failed")
		return "", err
	}

	// Get sender
	fromAccount, err := s.accountRepository.GetAccountForUpdateByUser(
		tx,
		req.FromAccountID,
		userID,
	)

	if err == sql.ErrNoRows {
		err := errors.New("sender account not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, "sender account not found")
		return "", err
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "sender lookup failed")
		return "", err
	}

	if fromAccount.Status != "active" {
		err := errors.New("sender account is not active")
		span.RecordError(err)
		span.SetStatus(codes.Error, "sender account inactive")
		return "", err
	}

	if fromAccount.Balance < req.Amount {
		err := errors.New("insufficient balance")
		span.RecordError(err)
		span.SetStatus(codes.Error, "insufficient balance")
		return "", err
	}

	// Get recipient
	toAccount, err := s.accountRepository.GetAccountForUpdate(
		tx,
		req.ToAccountID,
	)

	if err == sql.ErrNoRows {
		err := errors.New("recipient account not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, "recipient account not found")
		return "", err
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "recipient lookup failed")
		return "", err
	}

	if toAccount.Status != "active" {
		err := errors.New("recipient account is not active")
		span.RecordError(err)
		span.SetStatus(codes.Error, "recipient account inactive")
		return "", err
	}

	// Deduct sender
	err = s.accountRepository.DeductBalance(
		tx,
		req.FromAccountID,
		req.Amount,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "deduct balance failed")
		return "", err
	}

	// Add recipient
	err = s.accountRepository.AddBalance(
		tx,
		req.ToAccountID,
		req.Amount,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "add balance failed")
		return "", err
	}

	referenceNumber, err = GenerateReferenceNumber(12)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate reference failed")
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
		span.RecordError(err)
		span.SetStatus(codes.Error, "create transfer failed")
		return "", err
	}

	err = s.idempotencyRepository.Create(
		tx,
		idempotencyKey,
		referenceNumber,
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save idempotency failed")
		return "", err
	}

	// Commit
	if err := tx.Commit(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "commit failed")
		return "", err
	}

	// Mark span as successful
	span.SetStatus(codes.Ok, "transfer completed successfully")

	return transfer.ReferenceNumber, nil
}

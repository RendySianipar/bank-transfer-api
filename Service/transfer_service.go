package service

import (
	"bank-transfer-api/model"
	"bank-transfer-api/observability"
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

var tracer = otel.Tracer("bank-transfer-api/transfer")

type TransferService struct {
	db                    *sql.DB
	accountRepository     AccountRepositoryInterface
	idempotencyRepository IdempotencyRepositoryInterface
	metrics               *observability.Metrics
}

func NewTransferService(
	db *sql.DB,
	accountRepository AccountRepositoryInterface,
	idempotencyRepository IdempotencyRepositoryInterface,
	metrics *observability.Metrics,
) *TransferService {
	return &TransferService{
		db:                    db,
		accountRepository:     accountRepository,
		idempotencyRepository: idempotencyRepository,
		metrics:               metrics,
	}
}

func (s *TransferService) GetAllTransfer() ([]model.Transfer, error) {
	allTrf, err := s.accountRepository.GetAllTransfer()
	if err != nil {
		return nil, err
	}

	return allTrf, err
}

func (s *TransferService) Transfer(
	ctx context.Context,
	req model.TransferRequest,
	userID string,
	idempotencyKey string,
) (string, error) {
	// Record transfer duration
	start := time.Now()
	defer func() {
		s.metrics.TransferDuration.Record(
			context.Background(),
			time.Since(start).Seconds(),
		)
	}()

	// Create main span for this transfer operation
	ctx, span := tracer.Start(ctx, "transfer")
	defer span.End()

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
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "validation"),
			),
		)
		return "", err
	}

	if req.ToAccountID == "" {
		err := errors.New("to account ID is required")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "validation"),
			),
		)
		return "", err
	}

	if req.FromAccountID == req.ToAccountID {
		err := errors.New("sender and recepient cannot be the same")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "validation"),
			),
		)
		return "", err
	}

	// Validate amount
	if req.Amount <= 0 {
		err := errors.New("amount must be greater than 0")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "validation"),
			),
		)
		return "", err
	}

	if req.Amount > 10000000 {
		err := errors.New("amount exceeds maximum transfer limit")
		span.RecordError(err)
		span.SetStatus(codes.Error, "validation failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "validation"),
			),
		)
		return "", err
	}

	// Start transaction with child span
	var tx *sql.Tx
	err := spanOperation(ctx, "begin_transaction", func() error {
		var err error
		tx, err = s.db.BeginTx(ctx, nil)
		return err
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "transaction begin failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	defer tx.Rollback()

	// Check idempotency with child span
	var referenceNumber string
	err = spanOperation(ctx, "idempotency_lookup", func() error {
		var err error
		referenceNumber, err = s.idempotencyRepository.GetReferenceNumber(
			ctx,
			tx,
			idempotencyKey,
		)
		return err
	})

	if err == nil {
		// Idempotency hit - request was already processed
		s.metrics.IdempotencyHits.Add(
			ctx,
			1,
		)
		span.SetStatus(codes.Ok, "idempotency hit")
		return referenceNumber, nil
	}

	if err != sql.ErrNoRows {
		span.RecordError(err)
		span.SetStatus(codes.Error, "idempotency lookup failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	// Get sender account with child span
	var fromAccount *model.Account
	err = spanOperation(ctx, "lock_sender_account", func() error {
		var err error
		fromAccount, err = s.accountRepository.GetAccountForUpdateByUser(
			ctx,
			tx,
			req.FromAccountID,
			userID,
		)
		return err
	})

	if err == sql.ErrNoRows {
		err := errors.New("sender account not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, "sender account not found")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "sender_not_found"),
			),
		)
		return "", err
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "sender lookup failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	if fromAccount.Status != "active" {
		err := errors.New("sender account is not active")
		span.RecordError(err)
		span.SetStatus(codes.Error, "sender account inactive")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "inactive_account"),
			),
		)
		return "", err
	}

	if fromAccount.Balance < req.Amount {
		err := errors.New("insufficient balance")
		span.RecordError(err)
		span.SetStatus(codes.Error, "insufficient balance")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "insufficient_balance"),
			),
		)
		return "", err
	}

	// Get recipient account with child span
	var toAccount *model.Account
	err = spanOperation(ctx, "lock_recipient_account", func() error {
		var err error
		toAccount, err = s.accountRepository.GetAccountForUpdate(
			ctx,
			tx,
			req.ToAccountID,
		)
		return err
	})

	if err == sql.ErrNoRows {
		err := errors.New("recipient account not found")
		span.RecordError(err)
		span.SetStatus(codes.Error, "recipient account not found")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "recipient_not_found"),
			),
		)
		return "", err
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "recipient lookup failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	if toAccount.Status != "active" {
		err := errors.New("recipient account is not active")
		span.RecordError(err)
		span.SetStatus(codes.Error, "recipient account inactive")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "inactive_account"),
			),
		)
		return "", err
	}

	// Deduct balance with child span
	err = spanOperation(ctx, "deduct_balance", func() error {
		return s.accountRepository.DeductBalance(
			ctx,
			tx,
			req.FromAccountID,
			req.Amount,
		)
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "deduct balance failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	// Add balance with child span
	err = spanOperation(ctx, "add_balance", func() error {
		return s.accountRepository.AddBalance(
			ctx,
			tx,
			req.ToAccountID,
			req.Amount,
		)
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "add balance failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	referenceNumber, err = GenerateReferenceNumber(12)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "generate reference failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "generation_error"),
			),
		)
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

	// Create transfer record with child span
	err = spanOperation(ctx, "create_transfer", func() error {
		return s.accountRepository.CreateTransfer(
			ctx,
			tx,
			transfer,
		)
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "create transfer failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	// Save idempotency record with child span
	err = spanOperation(ctx, "save_idempotency", func() error {
		return s.idempotencyRepository.Create(
			ctx,
			tx,
			idempotencyKey,
			referenceNumber,
		)
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "save idempotency failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "database_error"),
			),
		)
		return "", err
	}

	// Commit transaction with child span
	err = spanOperation(ctx, "commit_transaction", func() error {
		return tx.Commit()
	})

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "commit failed")
		s.metrics.TransferFailures.Add(
			ctx,
			1,
			metric.WithAttributes(
				attribute.String("reason", "commit_error"),
			),
		)
		return "", err
	}

	// SUCCESS: Record successful transfer
	s.metrics.TransfersTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			attribute.String("status", "success"),
		),
	)

	span.SetStatus(codes.Ok, "transfer completed successfully")

	return transfer.ReferenceNumber, nil
}

// Helper function for creating child spans
func spanOperation(
	ctx context.Context,
	name string,
	fn func() error,
) error {
	ctx, span := tracer.Start(ctx, name)
	defer span.End()

	if err := fn(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetStatus(codes.Ok, "completed")
	return nil
}

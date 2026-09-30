package observability

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	TransfersTotal   metric.Int64Counter
	TransferFailures metric.Int64Counter
	IdempotencyHits  metric.Int64Counter
	TransferDuration metric.Float64Histogram
}

func NewMetrics(ctx context.Context) (*Metrics, error) {
	meter := otel.Meter("bank-transfer-api")

	transfersTotal, err := meter.Int64Counter(
		"bank_transfers_total",
		metric.WithDescription("Total number of transfer attempts"),
	)
	if err != nil {
		return nil, err
	}

	transferFailures, err := meter.Int64Counter(
		"bank_transfer_failures_total",
		metric.WithDescription("Total number of failed transfers"),
	)
	if err != nil {
		return nil, err
	}

	idempotencyHits, err := meter.Int64Counter(
		"bank_transfer_idempotency_hits_total",
		metric.WithDescription("Total number of idempotency cache hits"),
	)
	if err != nil {
		return nil, err
	}

	transferDuration, err := meter.Float64Histogram(
		"bank_transfer_duration_seconds",
		metric.WithDescription("Duration of transfer operations in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		TransfersTotal:   transfersTotal,
		TransferFailures: transferFailures,
		IdempotencyHits:  idempotencyHits,
		TransferDuration: transferDuration,
	}, nil
}

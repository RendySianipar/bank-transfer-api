package model

import "time"

type Account struct {
	ID        string `json:"id"`
	OwnerName string `json:"owner_name"`
	Balance   int64  `json:"balance"`
	Status    string `json:"status"`
}

type TransferRequest struct {
	ID            string `json:"id"`
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount        int64  `json:"amount"`
}

type Transfer struct {
	ID              string    `json:"id"`
	ReferenceNumber string    `json:"reference_number"`
	FromAccountID   string    `json:"from_account_id"`
	ToAccountID     string    `json:"to_account_id"`
	Amount          int64     `json:"amount"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

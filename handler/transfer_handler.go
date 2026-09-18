package handler

import (
	service "bank-transfer-api/Service"
	"bank-transfer-api/middleware"
	"bank-transfer-api/model"
	"encoding/json"
	"net/http"
)

func GetAllTransfer(service *service.TransferService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		allTrf, err := service.GetAllTransfer()

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allTrf)
	}
}

func TransferHandler(service *service.TransferService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req model.TransferRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		idempotencyKey := r.Header.Get("Idempotency-Key")

		if idempotencyKey == "" {
			http.Error(
				w,
				"Idempotency-Key header is required",
				http.StatusBadRequest,
			)
			return
		}

		userID, ok := r.Context().Value(middleware.UserIDKey).(string)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}

		referenceNumber, err := service.Transfer(req, userID, idempotencyKey)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"message":          "transfer successful",
			"reference_number": referenceNumber,
		})

	}

}

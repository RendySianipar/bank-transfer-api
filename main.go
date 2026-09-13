package main

import (
	repository "bank-transfer-api/Repository"
	service "bank-transfer-api/Service"
	"bank-transfer-api/database"
	"bank-transfer-api/handler"
	"fmt"
	"log"
	"net/http"
)

func main() {
	db, err := database.Connect()

	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	defer db.Close()

	accountRepository := repository.NewAccountRepository(db)
	idempotencyRepository := repository.NewIdempotencyRepository(db)

	transferService := service.NewTransferService(
		db,
		accountRepository,
		idempotencyRepository,
	)

	http.HandleFunc(
		"/transfer",
		handler.TransferHandler(transferService),
	)

	http.HandleFunc(
		"/getAllTransfer",
		handler.GetAllTransfer(transferService),
	)

	fmt.Println("server started at :7070")
	if err := http.ListenAndServe(":7070", nil); err != nil {
		log.Fatal(err)
	}

}

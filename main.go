package main

import (
	repository "bank-transfer-api/Repository"
	service "bank-transfer-api/Service"
	"bank-transfer-api/database"
	"bank-transfer-api/handler"
	"bank-transfer-api/middleware"
	"fmt"
	"log"
	"net/http"

	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatal("failed to connect database: ", err)
	}

	defer db.Close()

	// Auth
	userRepository := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepository)

	http.HandleFunc("/register", handler.RegisterHandler(authService))
	http.HandleFunc("/login", handler.LoginHandler(authService))

	// Transfer
	accountRepository := repository.NewAccountRepository(db)
	idempotencyRepository := repository.NewIdempotencyRepository(db)

	transferService := service.NewTransferService(
		db,
		accountRepository,
		idempotencyRepository,
	)

	http.HandleFunc("/transfer", middleware.JWTMiddleware(handler.TransferHandler(transferService)))

	http.HandleFunc(
		"/getAllTransfer", middleware.JWTMiddleware(
			handler.GetAllTransfer(transferService)),
	)

	fmt.Println("server started at :7070")
	if err := http.ListenAndServe(":7070", nil); err != nil {
		log.Fatal(err)
	}

}

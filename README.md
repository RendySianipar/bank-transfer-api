# Bank Transfer Api

A RESTful bank transfer API built with Go and MySQL

## Tech Stack

- Go
- net/http
- database/sql
- MySQL
- REST API
- UUID
- Database Transactions

## Architecture

HTTP Request
↓
Handler
↓
Service
↓
Repository
↓
MySQL

## Features

- Account validation
- Money transfer
- Database transactions
- Row-level locking
- Transfer history
- Transaction reference number
- Idempotency key
- Validation account, amount, and balance
- Money Handling
- Authentication and Authorization
- Concurrency Testing

## Project Structure

bank-transfer-api/
├── database/
├── handler/
├── model/
├── repository/
├── service/
├── main.go
└── README.md

## How to Run

1. Clone the repository
2. Install dependencies. go mod download
3. Configure environment variables. Create .env file.
4. Setup the database. database/schema.sql
5. Run the application. go run main.go

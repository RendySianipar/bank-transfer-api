CREATE DATABASE bank_transfer_db;

USE bank_transfer_db;

CREATE TABLE accounts (
    id VARCHAR(20) PRIMARY KEY,
    owner_name VARCHAR(100) NOT NULL,
    balance DECIMAL(15,2) NOT NULL,
    status ENUM('active', 'frozen') NOT NULL
);

CREATE TABLE transfers (
    id VARCHAR(36) PRIMARY KEY,
    reference_number VARCHAR(20) NOT NULL UNIQUE,
    from_account_id VARCHAR(20) NOT NULL,
    to_account_id VARCHAR(20) NOT NULL,
    amount DECIMAL(15,2) NOT NULL,
    status ENUM('success', 'failed') NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (from_account_id) REFERENCES accounts(id),
    FOREIGN KEY (to_account_id) REFERENCES accounts(id)
);

CREATE TABLE idempotency_keys (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    idempotency_key VARCHAR(100) NOT NULL UNIQUE,
    reference_number VARCHAR(20) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- Combined version of schema.sql + all migrations (000001, 000002, 000003),
-- in the correct dependency order, for MySQL's official Docker image to
-- run automatically via /docker-entrypoint-initdb.d/ on first container
-- startup (only when the data volume is empty - it will NOT re-run on
-- every restart).
--
-- NOTE: this duplicates content that also lives in schema.sql and
-- database/migrations/*.up.sql. That duplication exists because
-- docker-entrypoint-initdb.d requires everything needed in one pass, run
-- in alphabetical filename order - and "000001_..." would alphabetically
-- sort BEFORE "schema.sql", which would run a migration before the table
-- it modifies even exists. If you add a new migration going forward,
-- remember to also reflect it here, or this file will drift out of sync
-- with the real schema.
--
-- No CREATE DATABASE / USE here - MySQL's entrypoint already creates and
-- selects the database named by the MYSQL_DATABASE env var before running
-- this script.

CREATE TABLE accounts (
    id VARCHAR(20) PRIMARY KEY,
    owner_name VARCHAR(100) NOT NULL,
    balance BIGINT NOT NULL,
    status ENUM('active', 'frozen') NOT NULL,
    user_id VARCHAR(36) NOT NULL
);

CREATE TABLE transfers (
    id VARCHAR(36) PRIMARY KEY,
    reference_number VARCHAR(20) NOT NULL UNIQUE,
    from_account_id VARCHAR(20) NOT NULL,
    to_account_id VARCHAR(20) NOT NULL,
    amount BIGINT NOT NULL,
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

CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE accounts
ADD CONSTRAINT fk_accounts_user FOREIGN KEY (user_id) REFERENCES users(id);

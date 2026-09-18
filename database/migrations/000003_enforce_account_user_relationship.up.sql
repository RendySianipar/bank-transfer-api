ALTER TABLE accounts MODIFY COLUMN user_id VARCHAR(36) NOT NULL;

ALTER TABLE accounts ADD CONSTRAINT fk_accounts_user FOREIGN KEY (user_id) REFERENCES users(id);
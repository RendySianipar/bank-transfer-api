package repository

import (
	"database/sql"

	"bank-transfer-api/model"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user model.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (id, username, password_hash) VALUES (?, ?, ?)",
		user.ID, user.UserName, user.PasswordHash,
	)
	return err
}

func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User

	err := r.db.QueryRow(
		"SELECT id, username, password_hash FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.UserName, &user.PasswordHash)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

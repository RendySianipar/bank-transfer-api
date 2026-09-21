package service

import (
	"bank-transfer-api/model"
)

type fakeUserRepository struct {
	user model.User
}

func newfakeAuthRepository() *fakeUserRepository {
	return &fakeUserRepository{
		user: model.User{
			ID:           "",
			UserName:     "",
			PasswordHash: "",
		},
	}
}

// func (*fakeUserRepository) CreateUser(user model.User) error {

// }

// func (*fakeUserRepository) GetUserByUsername(username string) (*model.User, error) {

// }

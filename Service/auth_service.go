package service

import (
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	repository "bank-transfer-api/Repository"
	"bank-transfer-api/model"
	"bank-transfer-api/utils"
)

type AuthService struct {
	userRepository *repository.UserRepository
}

func NewAuthService(userRepository *repository.UserRepository) *AuthService {
	return &AuthService{userRepository: userRepository}
}

func (s *AuthService) Register(req model.RegisterRequest) error {
	if req.Username == "" || req.Password == "" {
		return errors.New("username and password are required")
	}

	if len(req.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	// bcrypt.GenerateFromPassword automatically adds a random salt,
	// so two users with the same password will still have DIFFERENT hashes.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := model.User{
		ID:           uuid.NewString(),
		UserName:     req.Username,
		PasswordHash: string(hashedPassword),
	}

	return s.userRepository.CreateUser(user)
}

func (s *AuthService) Login(req model.LoginRequest) (string, error) {
	user, err := s.userRepository.GetUserByUsername(req.Username)
	if err != nil {
		return "", errors.New("invalid username or password")
	}

	// CompareHashAndPassword is what "proves" the password matches,
	// WITHOUT ever needing to know or decode the original password.
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		return "", errors.New("invalid username or password")
	}

	token, err := utils.GenerateToken(user.ID, user.UserName)
	if err != nil {
		return "", err
	}

	return token, nil
}

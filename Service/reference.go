package service

import (
	"crypto/rand"
)

const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func GenerateReferenceNumber(length int) (string, error) {
	bytes := make([]byte, length)

	_, err := rand.Read(bytes)

	if err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[int(b)%len(charset)]
	}
	return "TRX-" + string(bytes), nil
}

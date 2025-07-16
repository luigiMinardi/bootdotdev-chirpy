package auth

import (
	"fmt"

	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	if len(password) < 1 {
		logging.LogError("password is empty: %s", password)
		return "", fmt.Errorf("password is empty")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		logging.LogError("failed to hash password: %s", err)
		return "", err
	}
	return string(hashedPassword), nil
}

func CheckPasswordHash(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		logging.LogError("hash and password comparison errored with: %s", err)
		return err
	}
	return nil
}

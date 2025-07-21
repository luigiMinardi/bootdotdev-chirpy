package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
	"golang.org/x/crypto/bcrypt"
)

const (
	TokenIssuerAPI string = "chirpy"
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

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    TokenIssuerAPI,
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	})

	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		logging.LogError("MakeJWT signedToken errored with: %s", err)
		return "", err
	}
	return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(t *jwt.Token) (any, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		logging.LogError("ValidateJWT parseWithClaims errored with: %s", err)
		return uuid.Nil, err
	}

	subject, err := token.Claims.GetSubject()
	if err != nil {
		logging.LogError("ValidateJWT Claims GetSubject errored with: %s", err)
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		logging.LogError("ValidateJWT Claims GetIssuer errored with: %s", err)
		return uuid.Nil, err
	}

	if issuer != TokenIssuerAPI {
		logging.LogError("ValidateJWT returned wrong issuer: %s", issuer)
		return uuid.Nil, fmt.Errorf("Issuer '%s' is not the API issuer '%s'.", issuer, TokenIssuerAPI)
	}

	uid, err := uuid.Parse(subject)
	if err != nil {
		logging.LogError("ValidateJWT UUID Parse errored with: %s", err)
		return uuid.Nil, err
	}
	return uid, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authorization := headers.Get("Authorization")
	if len(authorization) <= 0 {
		return "", fmt.Errorf("Authorization header with invalid length: %v", len(authorization))
	}
	tokenArr := strings.Split(authorization, " ")
	if len(tokenArr) < 2 {
		return "", fmt.Errorf("Token Malformed: %v", tokenArr)
	}
	return tokenArr[1], nil
}

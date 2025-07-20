package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
)

var tokenSecret = "secretTest"

func TestMakeAndValidateJWT(t *testing.T) {
	uid := uuid.New()
	token, err := MakeJWT(uid, tokenSecret, time.Duration(time.Second*30))
	if err != nil {
		logging.LogInfo("testing MakeJWT with UUID: %s", uid)
		t.Errorf("failed to MakeJWT: %s", err)
	}
	tokenUUID, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		logging.LogInfo("testing MakeJWT generated token: %s", token)
		t.Errorf("failed to ValidateJWT: %s", err)
	}
	if uid != tokenUUID {
		logging.LogInfo("testing MakeJWT with UUID: %s", uid)
		t.Errorf("testing ValidateJWT returned UUID: %s", tokenUUID)
	}
}

func TestMakeAndValidateAnExpiredJWT(t *testing.T) {
	uid := uuid.New()
	token, err := MakeJWT(uid, tokenSecret, time.Duration(time.Nanosecond))
	if err != nil {
		logging.LogInfo("testing MakeJWT with UUID: %s", uid)
		t.Errorf("failed to MakeJWT: %s", err)
	}
	tokenUUID, err := ValidateJWT(token, tokenSecret)
	if err == nil {
		logging.LogInfo("testing MakeJWT with UUID: %s", uid)
		logging.LogInfo("testing MakeJWT generated token: %s", token)
		t.Errorf("ValidateJWT worked with expired token returning UUID: %s", tokenUUID)
	}
	if uid == tokenUUID {
		t.Errorf("ValidateJWT returned valid UUID in an expired token: %s", token)
	}
}

func TestValidateAJWTWithNoUUID(t *testing.T) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Second * 30)),
		Subject:   "",
	})

	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		logging.LogError("invalid token data: %s", token)
		t.Errorf("invalid token signing failed with: %s", err)
	}

	_, err = ValidateJWT(signedToken, tokenSecret)
	if err == nil {
		t.Errorf("ValidateJWT worked with an invalid empty UUID%s", ".")
	}
}

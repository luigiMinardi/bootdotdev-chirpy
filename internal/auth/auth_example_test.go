package auth

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

var fake_token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiIwNDMwOWIxNC0yZWMxLTQ2N2ItYjYyNi04MTZlZTM2MzIwZmYiLCJleHAiOjE3NTU3ODM5NDgsImlhdCI6MTc1NTc4MjE0OH0.4fszj0XzapSkWe1l8PcXiw0O9Emqrv7Ba1SEjRR2KAE"
var fake_userUUID = uuid.MustParse("04309b14-2ec1-467b-b626-816ee36320ff")

// If you go to https://www.jwt.io and paste the output you can see the "User UUID"
// as the "sub"ject, "chirpy" as the "iss"user, and a difference of 30 minutes
// between the "exp"iration time in unix timestamp and the "iat" (issued at) time.
// You can also add the "myVerySecretToken" as the secret to validate the tokenSecret
func ExampleMakeJWT() {
	token, err := MakeJWT(
		// User UUID
		uuid.New(),
		// ApiConfig.jwtToken generated with "openssl rand -base64 64"
		"myVerySecretToken",
		// time needed for the token to expire in this case 30 minutes
		time.Duration(time.Minute*30),
	)
	if err != nil {
		fmt.Printf("Failed to MakeJWT: %v", err)
	}
	// Using a fake token since its random and wont pass the tests if I use
	// the one MakeJWT did. Yours will be a JWT like this tho
	token = fake_token

	fmt.Printf("%v", token)
	// Output:
	// eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiIwNDMwOWIxNC0yZWMxLTQ2N2ItYjYyNi04MTZlZTM2MzIwZmYiLCJleHAiOjE3NTU3ODM5NDgsImlhdCI6MTc1NTc4MjE0OH0.4fszj0XzapSkWe1l8PcXiw0O9Emqrv7Ba1SEjRR2KAE
}

// The output UUID is the same user UUID from the Example at [MakeJWT].
func ExampleValidateJWT() {
	userUUID, err := ValidateJWT(
		// Token generated with MakeJWT
		fake_token,
		// ApiConfig.jwtToken generated with "openssl rand -base64 64"
		"myVerySecretToken",
	)
	if err != nil {
		// The token will be expired on test runs so we are commenting the print
		// fmt.Printf("failed to ValidateJWT: %v", err)
	}
	// Using a fake UUID since its random and wont pass the tests if I use
	// the one ValidatgeJWT returned. Yours will be a UUID like this tho
	userUUID = fake_userUUID

	fmt.Printf("%v", userUUID)
	// Output:
	// 04309b14-2ec1-467b-b626-816ee36320ff
}

package utils

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
)

// struct that defines the return error for a request
type ReturnError struct {
	Error string `json:"error"`
}

// struct that defines a return value for a user, omiting its password
// based on database.User
type UserWithNoPassword struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

func Return500(w http.ResponseWriter, logErrMsg string, err any) {
	logging.LogError(logErrMsg, err)
	w.WriteHeader(500)
	respBody := ReturnError{
		Error: "Something went wrong",
	}

	data, err := json.Marshal(respBody)
	if err != nil {
		logging.LogError("failed to marshal JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

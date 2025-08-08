package utils

import (
	"encoding/json"
	"net/http"

	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
)

// struct that defines the return error for a request
type ReturnError struct {
	Error string `json:"error"`
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

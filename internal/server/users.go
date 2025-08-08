package server

import (
	"encoding/json"
	"net/http"

	"github.com/luigiMinardi/bootdotdev-chirpy/internal/auth"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/database"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/utils"
)

// POST /api/users
func (cfg *ApiConfig) PostUsersHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		logging.LogError("failed to decode params: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
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
		return
	}
	passwd, err := auth.HashPassword(params.Password)
	if err != nil {
		logging.LogError("Hash Password failed: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
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
		return
	}

	userParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: passwd,
	}
	user, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		logging.LogError("failed to create user: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
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
		return
	}
	respBody := utils.UserWithNoPassword{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}

	data, err := json.Marshal(respBody)
	if err != nil {
		logging.LogError("failed to marshal JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// PUT /api/users
func (cfg *ApiConfig) PutUsersHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		logging.LogError("failed to get token: %s", err)
		w.WriteHeader(401)
		respBody := utils.ReturnError{
			Error: "You're not logged in.",
		}
		data, err := json.Marshal(respBody)
		if err != nil {
			logging.LogError("failed to marshal JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
		return
	}

	id, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		logging.LogError("PUT /api/users failed to validate token: %s", err)
		w.WriteHeader(401)
		respBody := utils.ReturnError{
			Error: "Please log in again.",
		}
		data, err := json.Marshal(respBody)
		if err != nil {
			logging.LogError("failed to marshal JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		logging.LogError("failed to decode params: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
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
		return
	}
	passwd, err := auth.HashPassword(params.Password)
	if err != nil {
		logging.LogError("Hash Password failed: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
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
		return
	}

	userParams := database.UpdateUserParams{
		ID:             id,
		Email:          params.Email,
		HashedPassword: passwd,
	}
	user, err := cfg.db.UpdateUser(r.Context(), userParams)
	if err != nil {
		logging.LogError("failed to update user: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
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
		return
	}

	data, err := json.Marshal(user)
	if err != nil {
		logging.LogError("failed to marshal JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

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
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to decode params", err)
		return
	}
	passwd, err := auth.HashPassword(params.Password)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "Hash Password failed", err)
		return
	}

	userParams := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: passwd,
	}
	user, err := cfg.db.CreateUser(r.Context(), userParams)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to create user", err)
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
		utils.ResponseWithError(w, 401, "You're not logged in.", "failed to get token", err)
		return
	}

	id, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		utils.ResponseWithError(w, 401, "Please log in again.", "PUT /api/users failed to validate token", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to decode params", err)
		return
	}
	passwd, err := auth.HashPassword(params.Password)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "Hash Password failed", err)
		return
	}

	userParams := database.UpdateUserParams{
		ID:             id,
		Email:          params.Email,
		HashedPassword: passwd,
	}
	user, err := cfg.db.UpdateUser(r.Context(), userParams)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to update user", err)
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

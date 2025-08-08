package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/luigiMinardi/bootdotdev-chirpy/internal/auth"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/database"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/utils"
)

// POST /api/login
func (cfg *ApiConfig) LoginHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type returnVals struct {
		Id           string `json:"id"`
		CreatedAt    string `json:"created_at"`
		UpdatedAt    string `json:"updated_at"`
		Email        string `json:"email"`
		IsChirpyRed  bool   `json:"is_chirpy_red"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
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

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		logging.LogError("failed to retrieve user: %s", err)
		w.WriteHeader(401)
		respBody := utils.ReturnError{
			Error: "Incorrect email or password",
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

	err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		logging.LogError("failed to retrieve user: %s", err)
		w.WriteHeader(401)
		respBody := utils.ReturnError{
			Error: "Incorrect email or password",
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

	userJWT, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		logging.LogError("failed to generate user jwt: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
			Error: "Something wrong happened please contact the admin.",
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

	refreshTokenToken, err := auth.MakeRefreshToken()
	if err != nil {
		logging.LogError("refresh token failed to be generated: %s", err)
	}

	refreshTokenParams := database.CreateRefreshTokenParams{
		Token:     refreshTokenToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60), // expires in 2 months
	}
	refreshToken, err := cfg.db.CreateRefreshToken(r.Context(), refreshTokenParams)
	if err != nil {
		logging.LogError("failed to create refreshToken: %s", err)
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

	respBody := returnVals{
		Id:           user.ID.String(),
		CreatedAt:    user.CreatedAt.String(),
		UpdatedAt:    user.UpdatedAt.String(),
		Email:        user.Email,
		IsChirpyRed:  user.IsChirpyRed,
		Token:        userJWT,
		RefreshToken: refreshToken.Token,
	}

	data, err := json.Marshal(respBody)
	if err != nil {
		logging.LogError("failed to marshal JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// POST /api/refresh
func (cfg *ApiConfig) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		Token string `json:"token"`
	}

	refreshTokenToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		logging.LogError("failed to get refresh token: %s", err)
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
	refreshToken, err := cfg.db.GetRefreshToken(r.Context(), refreshTokenToken)
	if err != nil {
		logging.LogError("POST /api/refresh failed to find refresh token: %s", err)
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
	if time.Now().Compare(refreshToken.ExpiresAt) != -1 || refreshToken.RevokedAt.Valid == true {
		logging.LogError("refresh token expired org got revoked at: %s", refreshToken.RevokedAt.Time)
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
	token, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtSecret, time.Hour)
	if err != nil {
		logging.LogError("failed to generate user jwt: %s", err)
		w.WriteHeader(500)
		respBody := utils.ReturnError{
			Error: "Something wrong happened please contact the admin.",
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
	respBody := returnVals{
		Token: token,
	}
	data, err := json.Marshal(respBody)
	if err != nil {
		logging.LogError("failed to marshal JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// POST /api/revoke
func (cfg *ApiConfig) RevokeHandler(w http.ResponseWriter, r *http.Request) {
	refreshTokenToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		logging.LogError("failed to get refresh token: %s", err)
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
	err = cfg.db.RevokeRefreshToken(r.Context(), refreshTokenToken)
	if err != nil {
		logging.LogError("POST /api/revoke failed to find refresh token: %s", err)
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
	w.WriteHeader(204)
}

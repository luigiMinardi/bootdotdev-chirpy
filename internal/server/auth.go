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
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to decode params", err)
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		utils.ResponseWithError(w, 401, "Incorrect email or password", "failed to retrieve user", err)
		return
	}

	err = auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		utils.ResponseWithError(w, 401, "Incorrect email or password", "failed to retrieve user", err)
		return
	}

	userJWT, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something wrong happened please contact the admin.", "failed to generate user jwt", err)
		return
	}

	refreshTokenToken, err := auth.MakeRefreshToken()
	if err != nil {
		logging.LogError("refresh token failed to be generated", err)
	}

	refreshTokenParams := database.CreateRefreshTokenParams{
		Token:     refreshTokenToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60), // expires in 2 months
	}
	refreshToken, err := cfg.db.CreateRefreshToken(r.Context(), refreshTokenParams)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to create refreshToken", err)
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

	utils.ResponseWithJson(w, 200, respBody)
}

// POST /api/refresh
func (cfg *ApiConfig) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	type returnVals struct {
		Token string `json:"token"`
	}

	refreshTokenToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.ResponseWithError(w, 401, "You're not logged in.", "failed to get refresh token", err)
		return
	}
	refreshToken, err := cfg.db.GetRefreshToken(r.Context(), refreshTokenToken)
	if err != nil {
		utils.ResponseWithError(w, 401, "Please log in again.", "POST /api/refresh failed to find refresh token", err)
		return
	}
	if time.Now().Compare(refreshToken.ExpiresAt) != -1 || refreshToken.RevokedAt.Valid == true {
		utils.ResponseWithError(w, 401, "Please log in again.", "refresh token expired or got revoked at", err)
		return
	}
	token, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtSecret, time.Hour)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something wrong happened please contact the admin.", "failed to generate user jwt", err)
		return
	}
	respBody := returnVals{
		Token: token,
	}
	utils.ResponseWithJson(w, 200, respBody)
}

// POST /api/revoke
func (cfg *ApiConfig) RevokeHandler(w http.ResponseWriter, r *http.Request) {
	refreshTokenToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.ResponseWithError(w, 401, "You're not logged in.", "failed to get refresh token", err)
		return
	}
	err = cfg.db.RevokeRefreshToken(r.Context(), refreshTokenToken)
	if err != nil {
		utils.ResponseWithError(w, 401, "Please log in again.", "POST /api/revoke failed to find refresh token", err)
		return
	}
	w.WriteHeader(204)
}

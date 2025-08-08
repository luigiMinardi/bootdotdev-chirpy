package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/auth"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/database"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/utils"
)

// POST /api/chirps
func (cfg *ApiConfig) PostChirpsHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type returnVals struct {
		Id        string `json:"id"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Body      string `json:"body"`
		UserID    string `json:"user_id"`
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.ResponseWithError(w, 401, "You're not logged in.", "failed to get token", err)
		return
	}

	id, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		utils.ResponseWithError(w, 401, "Please log in again.", "POST /api/chirps failed to validate token", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to decode params", err)
		return
	}
	if len(params.Body) > 140 {
		utils.ResponseWithError(w, 400, "Chirp is too long", "chirp is too long", err)
		return
	}
	if params.Body == "" {
		utils.ResponseWithError(w, 400, "Empty \"body\" field", "empty \"body\" field", err)
		return
	}

	words := strings.Split(params.Body, " ")
	for wordIndex := range words {
		if strings.ToLower(words[wordIndex]) == "kerfuffle" {
			words[wordIndex] = "****"
			continue
		}
		if strings.ToLower(words[wordIndex]) == "sharbert" {
			words[wordIndex] = "****"
			continue
		}
		if strings.ToLower(words[wordIndex]) == "fornax" {
			words[wordIndex] = "****"
			continue
		}
	}

	params.Body = strings.Join(words, " ")

	chirpParams := database.CreateChirpParams{
		Body:   params.Body,
		UserID: id,
	}
	chirp, err := cfg.db.CreateChirp(r.Context(), chirpParams)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to create chirp", err)
		return
	}
	respBody := returnVals{
		Id:        chirp.ID.String(),
		CreatedAt: chirp.CreatedAt.String(),
		UpdatedAt: chirp.UpdatedAt.String(),
		Body:      chirp.Body,
		UserID:    chirp.UserID.String(),
	}

	data, err := json.Marshal(respBody)
	if err != nil {
		logging.LogError("failed to marshal JSON", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(201)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// GET /api/chirps
func (cfg *ApiConfig) GetChirpsHandler(w http.ResponseWriter, r *http.Request) {
	authorId := r.URL.Query().Get("author_id")
	sort := r.URL.Query().Get("sort")

	var chirps []database.Chirp

	if sort != "asc" && sort != "desc" {
		logging.LogInfo("sort", sort)
		sort = "asc"
	}

	if authorId != "" {
		authorUid, err := uuid.Parse(authorId)
		if err != nil {
			utils.ResponseWithError(w, 404, "Invalid Author ID", "invalid authorId", err)
			return
		}
		params := database.GetAllChirpsFromUserParams{
			UserID:    authorUid,
			SortOrder: sort,
		}
		chirps, err = cfg.db.GetAllChirpsFromUser(r.Context(), params)
		if err != nil {
			utils.ResponseWithError(w, 500, "Something went wrong", "failed to retrieve chirps", err)
			return
		}
	}
	chirps, err := cfg.db.GetAllChirps(r.Context(), sort)
	if err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to retrieve chirps", err)
		return
	}

	data, err := json.Marshal(chirps)
	if err != nil {
		logging.LogError("failed to marshal JSON", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// GET /api/chirps/{chirpID}
func (cfg *ApiConfig) GetChirpsByIdHandler(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("chirpID")

	id, err := uuid.Parse(idString)
	if err != nil {
		utils.ResponseWithError(w, 400, "Invaid \"chirpID\" path parameter", "failed to get uuid", err)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), id)
	if err != nil {
		utils.ResponseWithError(w, 404, "This chirp was deleted or don't exist", "failed to retrieve chirp", err)
		return
	}

	data, err := json.Marshal(chirp)
	if err != nil {
		logging.LogError("failed to marshal JSON", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// DELETE /api/chirps/{chirpID}
func (cfg *ApiConfig) DeleteChirpsByIdHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		utils.ResponseWithError(w, 401, "You're not logged in.", "failed to get token", err)
		return
	}

	userId, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		utils.ResponseWithError(w, 401, "Please log in again.", "PUT /api/users failed to validate token", err)
		return
	}

	idString := r.PathValue("chirpID")

	chirpId, err := uuid.Parse(idString)
	if err != nil {
		utils.ResponseWithError(w, 400, "Invaid \"chirpID\" path parameter", "failed to get uuid", err)
		return
	}

	dta := database.DeleteChirpParams{
		UserID: userId,
		ID:     chirpId,
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpId)
	if err != nil {
		utils.ResponseWithError(w, 404, "This chirp was deleted or don't exist", "failed to retrieve chirp", err)
		return
	}
	if chirp.UserID != userId {
		w.WriteHeader(403)
		return
	}

	deletedChirp, err := cfg.db.DeleteChirp(r.Context(), dta)
	if err != nil {
		utils.ResponseWithError(w, 404, "This chirp was deleted or don't exist", "failed to retrieve chirp", err)
		return
	}
	logging.LogInfo("removed", deletedChirp)

	w.WriteHeader(204)
	w.Header().Set("Content-Type", "application/json")
}

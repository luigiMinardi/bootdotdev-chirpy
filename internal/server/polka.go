package server

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/auth"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/utils"
)

// POST /api/polka/webhooks
func (cfg *ApiConfig) PolkaWebhookHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID uuid.UUID `json:"user_id"`
		} `json:"data"`
	}
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		utils.ResponseWithError(w, 401, "You are not authenticated", "failed to get api key", err)
		return
	}

	if apiKey != cfg.polkaKey {
		utils.ResponseWithError(w, 401, "You are not authenticated", "POST /api/polka/webhooks failed to validate api key", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		utils.ResponseWithError(w, 500, "Something went wrong", "failed to decode params", err)
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	_, err = cfg.db.UpgradeUserToChirpyRedByID(r.Context(), params.Data.UserID)
	if err != nil {
		utils.ResponseWithError(w, 404, "This user was deleted or don't exist", "failed to retrieve user", err)
		return
	}

	w.WriteHeader(204)
}

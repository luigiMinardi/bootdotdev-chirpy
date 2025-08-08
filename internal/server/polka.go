package server

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/auth"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
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
		logging.LogError("failed to get api key: %s", err)
		w.WriteHeader(401)
		respBody := utils.ReturnError{
			Error: "You are not authenticated",
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
	logging.LogInfo("apiKey: %s", apiKey)
	logging.LogInfo("cfg apiKey: %s", cfg.polkaKey)

	if apiKey != cfg.polkaKey {
		logging.LogError("POST /api/polka/webhooks failed to validate api key: %s", apiKey)
		w.WriteHeader(401)
		respBody := utils.ReturnError{
			Error: "You are not authenticated",
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

	if params.Event != "user.upgraded" {
		w.WriteHeader(204)
		return
	}

	_, err = cfg.db.UpgradeUserToChirpyRedByID(r.Context(), params.Data.UserID)
	if err != nil {
		logging.LogError("failed to retrieve user: %s", err)
		w.WriteHeader(404)
		respBody := utils.ReturnError{
			Error: "This user was deleted or don't exist",
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

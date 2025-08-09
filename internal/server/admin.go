package server

import (
	"fmt"
	"net/http"

	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
)

// endpoint to visualize the utils.ApiConfig.fileServerHits metric in html.
func (cfg *ApiConfig) endpointMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	_, err := w.Write(fmt.Appendf([]byte{}, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load()))
	if err != nil {
		logging.LogError("/metrics failed to write with error", err)
	}
}

// endpoint to reset the utils.ApiConfig related things on dev environment.
func (cfg *ApiConfig) endpointReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		w.Write([]byte("You can only reset on dev environment."))
		return
	}
	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		logging.LogError("failed to delete users", err)
		w.WriteHeader(500)
		w.Write([]byte("failed to reset db with err: " + err.Error()))
		return
	}
	err = cfg.db.DeleteAllChirps(r.Context())
	if err != nil {
		logging.LogError("failed to delete chirps", err)
		w.WriteHeader(500)
		w.Write([]byte("failed to reset db with err: " + err.Error()))
		return
	}
	logging.LogInfo("users reset at env", cfg.platform)
	w.WriteHeader(200)
	cfg.fileServerHits.Store(0)
	w.Write([]byte("fileServerHits reset to 0 and database reset to initial state."))
}

package server

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/database"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
)

// struct that holds api data like metrics environments, db etc.
type ApiConfig struct {
	// metric that counts how many times all endpoints that use it have been hit
	fileServerHits atomic.Int32
	// if you're in prod or dev environment
	platform string
	// data base
	db *database.Queries
	// jwt secret
	jwtSecret string
	// polka key
	polkaKey string
}

// Middleware function that counts how many times an endpoint has been hit, it
// does not save it so when server resets it's restarted.
func (cfg *ApiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		logging.LogInfo("current cfg.fileServerHits: %v", cfg.fileServerHits.Load())
		next.ServeHTTP(w, r)
	})
}

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
		logging.LogError("/metrics failed to write with error: %v\n", err)
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
	logging.LogInfo("users reset at env: %s", cfg.platform)
	w.WriteHeader(200)
	cfg.fileServerHits.Store(0)
	w.Write([]byte("fileServerHits reset to 0 and database reset to initial state."))
}

func NewServer() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Panicf(logging.LOGERROR + "DB_URL must be set")
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Panicf(logging.LOGERROR + "PLATFORM must be set")
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Panicf(logging.LOGERROR + "JWT_SECRET must be set")
	}
	polkaKey := os.Getenv("POLKA_KEY")
	if polkaKey == "" {
		log.Panicf(logging.LOGERROR + "POLKA_KEY must be set")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Panicf(logging.LOGERROR+"db connection failed with err: %v", err)
	}
	dbQueries := database.New(db)
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	apiCfg := &ApiConfig{}
	apiCfg.platform = platform
	apiCfg.db = dbQueries
	apiCfg.jwtSecret = jwtSecret
	apiCfg.polkaKey = polkaKey

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			logging.LogError("/healthz failed to write with error: %v\n", err)
		}
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.endpointMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.endpointReset)

	mux.HandleFunc("POST /api/chirps", apiCfg.PostChirpsHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.GetChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.GetChirpsByIdHandler)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.DeleteChirpsByIdHandler)

	mux.HandleFunc("POST /api/users", apiCfg.PostUsersHandler)
	mux.HandleFunc("PUT /api/users", apiCfg.PutUsersHandler)

	mux.HandleFunc("POST /api/login", apiCfg.LoginHandler)
	mux.HandleFunc("POST /api/refresh", apiCfg.RefreshHandler)
	mux.HandleFunc("POST /api/revoke", apiCfg.RevokeHandler)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.PolkaWebhookHandler)

	logging.LogInfo("HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logging.LogError("HTTP Server ListenAndServe error: %v\n", err)
	}
}

package server

import (
	"database/sql"
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
	// jwt secret generated with "openssl rand -base64 64"
	jwtSecret string
	// polka key
	polkaKey string
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

	mux.Handle("/app/", apiCfg.MiddlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		logging.LogInfo("method", r.Method)

		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			logging.LogError("/healthz failed to write with error", err)
		}
	})

	mux.HandleFunc("GET /admin/metrics", apiCfg.endpointMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.endpointReset)

	mux.Handle("POST /api/chirps", apiCfg.MiddlewareValidateJWT(apiCfg.PostChirpsHandler))
	mux.HandleFunc("GET /api/chirps", apiCfg.GetChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.GetChirpsByIdHandler)
	mux.Handle("DELETE /api/chirps/{chirpID}", apiCfg.MiddlewareValidateJWT(apiCfg.DeleteChirpsByIdHandler))

	mux.HandleFunc("POST /api/users", apiCfg.PostUsersHandler)
	mux.Handle("PUT /api/users", apiCfg.MiddlewareValidateJWT(apiCfg.PutUsersHandler))

	mux.HandleFunc("POST /api/login", apiCfg.LoginHandler)
	mux.HandleFunc("POST /api/refresh", apiCfg.RefreshHandler)
	mux.HandleFunc("POST /api/revoke", apiCfg.RevokeHandler)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.PolkaWebhookHandler)

	log.Printf(logging.LOGINFO+"HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logging.LogError("HTTP Server ListenAndServe error", err)
	}
}

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/database"
)

const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorReset  = "\033[0m"
	LogError    = ColorRed + "ERROR: " + ColorReset
	LogWarn     = ColorYellow + "WARN: " + ColorReset
	LogInfo     = ColorBlue + "INFO: " + ColorReset
)

// Logs (s) to the terminal with (arg) arguments, before (s) you have "INFO: "
// printed in Blue
func logInfo(s string, arg any) {
	log.Printf(LogInfo+s, arg)
}

// Logs (s) to the terminal with (arg) arguments, before (s) you have "ERROR: "
// printed in Red
func logError(s string, arg any) {
	log.Printf(LogError+s, arg)
}

// struct that holds api data like metrics environments, db etc.
type apiConfig struct {
	// metric that counts how many times all endpoints that use it have been hit
	fileServerHits atomic.Int32
	// if you're in prod or dev environment
	platform string
	// data base
	db *database.Queries
}

// Middleware function that counts how many times an endpoint has been hit, it
// does not save it so when server resets it's restarted.
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		logInfo("current cfg.fileServerHits: %v", cfg.fileServerHits.Load())
		next.ServeHTTP(w, r)
	})
}

// endpoint to visualize the apiConfig.fileServerHits metric in html.
func (cfg *apiConfig) endpointMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)

	_, err := w.Write(fmt.Appendf([]byte{}, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileServerHits.Load()))
	if err != nil {
		logError("/metrics failed to write with error: %v\n", err)
	}
}

// endpoint to reset the apiConfig related things on dev environment.
func (cfg *apiConfig) endpointReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(403)
		w.Write([]byte("You can only reset on dev environment."))
		return
	}
	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		logError("failed to delete users", err)
		w.WriteHeader(500)
		w.Write([]byte("failed to reset db with err: " + err.Error()))
		return
	}
	err = cfg.db.DeleteAllChirps(r.Context())
	if err != nil {
		logError("failed to delete chirps", err)
		w.WriteHeader(500)
		w.Write([]byte("failed to reset db with err: " + err.Error()))
		return
	}
	logInfo("users reset at env: %s", cfg.platform)
	w.WriteHeader(200)
	cfg.fileServerHits.Store(0)
	w.Write([]byte("fileServerHits reset to 0 and database reset to initial state."))
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Panicf(LogError + "DB_URL must be set")
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Panicf(LogError + "PLATFORM must be set")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Panicf(LogError+"db connection failed with err: %v", err)
	}
	dbQueries := database.New(db)
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	apiCfg := &apiConfig{}
	apiCfg.platform = platform
	apiCfg.db = dbQueries

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			logError("/healthz failed to write with error: %v\n", err)
		}
	})
	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body   string `json:"body"`
			UserID string `json:"user_id"`
		}
		type returnVals struct {
			Id        string `json:"id"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			Body      string `json:"body"`
			UserID    string `json:"user_id"`
		}
		type returnError struct {
			Error string `json:"error"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		if err := decoder.Decode(&params); err != nil {
			logError("failed to decode params: %s", err)
			w.WriteHeader(500)
			respBody := returnError{
				Error: "Something went wrong",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				logError("failed to marshal JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		if len(params.Body) > 140 {
			w.WriteHeader(400)
			respBody := returnError{
				Error: "Chirp is too long",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				logError("failed to marshal JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		if params.Body == "" {
			w.WriteHeader(400)
			respBody := returnError{
				Error: "Empty \"body\" field",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				logError("failed to marshal JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}

		logInfo("word: %s", params.Body)
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
		id, err := uuid.Parse(params.UserID)
		if err != nil {
			logError("failed to get uuid: %s", err)
			respBody := returnError{
				Error: "Invaid \"user_id\" field",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				logError("failed to marshal JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		nid := uuid.NullUUID{
			UUID:  id,
			Valid: true,
		}
		chirpParams := database.CreateChirpParams{
			Body:   params.Body,
			UserID: nid,
		}
		chirp, err := apiCfg.db.CreateChirp(r.Context(), chirpParams)
		if err != nil {
			logError("failed to create user: %s", err)
			w.WriteHeader(500)
			respBody := returnError{
				Error: "Something went wrong",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				logError("failed to marshal JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		respBody := returnVals{
			Id:        chirp.ID.String(),
			CreatedAt: chirp.CreatedAt.String(),
			UpdatedAt: chirp.UpdatedAt.String(),
			Body:      chirp.Body,
			UserID:    chirp.UserID.UUID.String(),
		}

		data, err := json.Marshal(respBody)
		if err != nil {
			logError("failed to marshal JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(201)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email string `json:"email"`
		}
		type returnVals struct {
			Id        string `json:"id"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
			Email     string `json:"email"`
		}
		type returnError struct {
			Error string `json:"error"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		if err := decoder.Decode(&params); err != nil {
			logError("failed to decode params: %s", err)
			w.WriteHeader(500)
			respBody := returnError{
				Error: "Something went wrong",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				logError("failed to marshal JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		user, err := apiCfg.db.CreateUser(r.Context(), params.Email)
		if err != nil {
			logError("failed to create user: %s", err)
			w.WriteHeader(500)
			respBody := returnError{
				Error: "Something went wrong",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				logError("failed to marshal JSON: %s", err)
				w.WriteHeader(500)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		respBody := returnVals{
			Id:        user.ID.String(),
			CreatedAt: user.CreatedAt.String(),
			UpdatedAt: user.UpdatedAt.String(),
			Email:     user.Email,
		}

		data, err := json.Marshal(respBody)
		if err != nil {
			logError("failed to marshal JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(201)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.endpointMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.endpointReset)

	logInfo("HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logError("HTTP Server ListenAndServe error: %v\n", err)
	}
}

package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/auth"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/database"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/utils"
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

// struct that defines a return value for a user, omiting its password
// based on database.User
type userWithNoPassword struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
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
	godotenv.Load()
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
	mux.HandleFunc("POST /api/chirps", apiCfg.PostChirpsHandler)
	mux.HandleFunc("GET /api/chirps", apiCfg.GetChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.GetChirpsByIdHandler)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.DeleteChirpsByIdHandler)

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
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
		passwd, err := auth.HashPassword(params.Password)
		if err != nil {
			logging.LogError("Hash Password failed: %s", err)
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

		userParams := database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: passwd,
		}
		user, err := apiCfg.db.CreateUser(r.Context(), userParams)
		if err != nil {
			logging.LogError("failed to create user: %s", err)
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
		respBody := userWithNoPassword{
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
	})
	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			logging.LogError("failed to get token: %s", err)
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

		id, err := auth.ValidateJWT(token, apiCfg.jwtSecret)
		if err != nil {
			logging.LogError("PUT /api/users failed to validate token: %s", err)
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
		passwd, err := auth.HashPassword(params.Password)
		if err != nil {
			logging.LogError("Hash Password failed: %s", err)
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

		userParams := database.UpdateUserParams{
			ID:             id,
			Email:          params.Email,
			HashedPassword: passwd,
		}
		user, err := apiCfg.db.UpdateUser(r.Context(), userParams)
		if err != nil {
			logging.LogError("failed to update user: %s", err)
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

		data, err := json.Marshal(user)
		if err != nil {
			logging.LogError("failed to marshal JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
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

		user, err := apiCfg.db.GetUserByEmail(r.Context(), params.Email)
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

		userJWT, err := auth.MakeJWT(user.ID, apiCfg.jwtSecret, time.Hour)
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
		refreshToken, err := apiCfg.db.CreateRefreshToken(r.Context(), refreshTokenParams)
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
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
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
		refreshToken, err := apiCfg.db.GetRefreshToken(r.Context(), refreshTokenToken)
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
		token, err := auth.MakeJWT(refreshToken.UserID, apiCfg.jwtSecret, time.Hour)
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
	})
	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
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
		err = apiCfg.db.RevokeRefreshToken(r.Context(), refreshTokenToken)
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
	})
	mux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
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
		logging.LogInfo("apiCfg apiKey: %s", apiCfg.polkaKey)

		if apiKey != apiCfg.polkaKey {
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

		_, err = apiCfg.db.UpgradeUserToChirpyRedByID(r.Context(), params.Data.UserID)
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
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.endpointMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.endpointReset)

	logging.LogInfo("HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logging.LogError("HTTP Server ListenAndServe error: %v\n", err)
	}
}

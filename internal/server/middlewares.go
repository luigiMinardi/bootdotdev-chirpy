package server

import (
	"context"
	"net/http"

	"github.com/luigiMinardi/bootdotdev-chirpy/internal/auth"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/logging"
	"github.com/luigiMinardi/bootdotdev-chirpy/internal/utils"
)

// Middleware function that counts how many times an endpoint has been hit, it
// does not save it so when server resets it's restarted.
func (cfg *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		logging.LogInfo("current cfg.fileServerHits", cfg.fileServerHits.Load())
		next.ServeHTTP(w, r)
	})
}

// Middleware function that validates JWT
func (cfg *ApiConfig) MiddlewareValidateJWT(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			utils.ResponseWithError(w, 401, "You're not logged in.", "failed to get token", err)
			return
		}

		id, err := auth.ValidateJWT(token, cfg.jwtSecret)
		if err != nil {
			utils.ResponseWithError(w, 401, "You're not logged in.", "failed to validate token", err)
			return
		}
		// adding the parsed id from the jwt to the Context so that it can be accessed by the HandleFunc's
		// This is a shallow copy of request so it only changes r.Context
		r = r.WithContext(context.WithValue(r.Context(), "id", id))
		next.ServeHTTP(w, r)
	})
}

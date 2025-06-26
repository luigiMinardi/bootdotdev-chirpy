package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
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

// struct that holds api data like metrics.
type apiConfig struct {
	// metric that counts how many times all endpoints that use it have been hit
	fileServerHits atomic.Int32
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

// endpoint to reset the apiConfig.fileServerHits metric to 0.
func (cfg *apiConfig) endpointReset(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	cfg.fileServerHits.Store(0)
}

func main() {
	mux := http.NewServeMux()
	srv := &http.Server{
		Handler: mux,
		Addr:    ":8080",
	}
	apiCfg := &apiConfig{}

	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			logError("/healthz failed to write with error: %v\n", err)
		}
	})
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}
		type returnVals struct {
			Error       string `json:"error,omitempty"`
			CleanedBody string `json:"cleaned_body,omitempty"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		if err := decoder.Decode(&params); err != nil {
			logError("failed to decode params: %s", err)
			w.WriteHeader(500)
			respBody := returnVals{
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
			respBody := returnVals{
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
			respBody := returnVals{
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
			logInfo("word: %s", words[wordIndex])
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

		w.WriteHeader(200)
		respBody := returnVals{
			CleanedBody: params.Body,
		}
		data, err := json.Marshal(respBody)
		if err != nil {
			logError("failed to marshal JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.endpointMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.endpointReset)

	log.Printf("HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		logError("HTTP Server ListenAndServe error: %v\n", err)
	}
}

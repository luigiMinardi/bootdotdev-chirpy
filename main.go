package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

type apiConfig struct {
	fileServerHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		log.Printf(LogInfo+"current cfg.fileServerHits: %v", cfg.fileServerHits.Load())
		next.ServeHTTP(w, r)
	})
}

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
		log.Printf(LogError+"/metrics failed to write with error: %v\n", err)
	}
}

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
			log.Printf(LogError+"/healthz failed to write with error: %v\n", err)
		}
	})
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}
		type returnVals struct {
			Error string `json:"error,omitempty"`
			Valid bool   `json:"valid,omitempty"`
		}
		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		if err := decoder.Decode(&params); err != nil {
			log.Printf(LogError+"failed to decode params: %s", err)
			w.WriteHeader(500)
			respBody := returnVals{
				Error: "Something went wrong",
			}
			data, err := json.Marshal(respBody)
			if err != nil {
				log.Printf(LogError+"failed to marshal JSON: %s", err)
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
				w.WriteHeader(500)
				log.Printf(LogError+"failed to marshal JSON: %s", err)
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
				w.WriteHeader(500)
				log.Printf(LogError+"failed to marshal JSON: %s", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(data)
			return
		}
		w.WriteHeader(200)
		respBody := returnVals{
			Valid: true,
		}
		data, err := json.Marshal(respBody)
		if err != nil {
			w.WriteHeader(500)
			log.Printf(LogError+"failed to marshal JSON: %s", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})
	mux.HandleFunc("GET /admin/metrics", apiCfg.endpointMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.endpointReset)

	log.Printf("HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf(LogError+"HTTP Server ListenAndServe error: %v\n", err)
	}
}

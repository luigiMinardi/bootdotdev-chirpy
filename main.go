package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	var srv http.Server
	srv.Handler = mux
	srv.Addr = ":8080"

	mux.Handle("/app/assets/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Printf("helthz failed to write with error: %v", err)
		}
	})

	log.Printf("HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("HTTP Server ListenAndServe error: %v\n", err)
	}
}

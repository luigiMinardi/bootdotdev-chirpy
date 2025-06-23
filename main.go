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

	log.Printf("HTTP server started on http://localhost%v\n", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("HTTP Server ListenAndServe error: %v\n", err)
	}
}

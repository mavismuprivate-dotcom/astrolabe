package main

import (
	"log"
	"net/http"
	"os"

	"astrolabe/internal/api"
	"astrolabe/internal/astrology"
)

func main() {
	resolver := astrology.NewCityResolver()
	svc := astrology.NewService(resolver)
	apiHandler := api.NewHandler(svc)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	mux.Handle("/healthz", apiHandler)
	mux.Handle("/", http.FileServer(http.Dir("web")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	log.Printf("astrolabe server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"astrolabe/internal/api"
	"astrolabe/internal/astrology"
	"astrolabe/internal/storage"
)

func main() {
	resolver := astrology.NewCityResolver()
	svc := astrology.NewService(resolver)

	dbPath := os.Getenv("ASTROLABE_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "astrolabe.db")
	}

	reportStore, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize report store: %v", err)
	}
	defer func() {
		if err := reportStore.Close(); err != nil {
			log.Printf("failed to close report store: %v", err)
		}
	}()

	apiHandler := api.NewHandlerWithStore(svc, reportStore)

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

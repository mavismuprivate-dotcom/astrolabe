package main

import (
	"log"
	"net/http"

	"astrolabe/internal/api"
	"astrolabe/internal/astrology"
	"astrolabe/internal/storage"
)

func main() {
	cfg, err := loadConfigFromEnv()
	if err != nil {
		log.Fatalf("invalid runtime configuration: %v", err)
	}

	resolver := astrology.NewCityResolver()
	svc := astrology.NewService(resolver)

	reportStore, err := storage.NewSQLiteStore(cfg.DBPath)
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

	logger := log.New(log.Writer(), "", log.LstdFlags)
	addr := ":" + cfg.Port
	log.Printf("astrolabe server listening on %s", addr)
	if err := http.ListenAndServe(addr, withRuntimeMiddleware(logger, cfg.RateLimitPerMinute, mux)); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

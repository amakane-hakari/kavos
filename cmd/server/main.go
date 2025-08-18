package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	apphttp "github.com/amakane-hakari/kavos/internal/api/http"
	"github.com/amakane-hakari/kavos/internal/store"
)

func main() {
	addr := getEnv("KAVOS_HTTP_ADDR", ":8080")

	st := store.New[string, string]()

	router := apphttp.NewRouter(st)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("starting server on %s", addr)
	
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
		case <-ctx.Done():
			log.Println("shutdown signal received")
		case err := <-errCh:
			log.Printf("server error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	} else {
		log.Println("server stopped")
	}
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
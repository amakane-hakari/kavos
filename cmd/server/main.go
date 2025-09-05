// Package main は KVSのメインエントリポイントです。
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
	ilog "github.com/amakane-hakari/kavos/internal/log"
	"github.com/amakane-hakari/kavos/internal/metrics"
	"github.com/amakane-hakari/kavos/internal/store"
)

func main() {
	addr := getEnv("KAVOS_HTTP_ADDR", ":8080")

	logger := ilog.New()

	var mx metrics.Interface
	if os.Getenv("METRICS") == "prometheus" {
		mx = metrics.NewProm("kavos")
	} else {
		mx = metrics.NewSimple()
	}
	st := store.New[string, string](
		store.WithShards(16),
		store.WithCleanupInterval(1*time.Second),
		store.WithLogger(logger),
		store.WithMetrics(mx),
	).WithEvictor(store.NewLRUEvictor[string, string](10000))

	router := apphttp.NewRouter(st, logger)

	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("server.start addr=%s pid=%d", addr, os.Getpid())

	sigCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-sigCtx.Done():
		log.Printf("server.stop signal received: %v", sigCtx.Err())
	case err := <-errCh:
		log.Printf("server.listen.error err=%v", err)
	}

	// Draining 開始(ヘルスチェックを503にする)
	apphttp.SetDraining(true)

	// シャットダウン待機時間
	shutdownTimeout := 10 * time.Second
	if v := os.Getenv("SHUTDOWN_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			shutdownTimeout = d
		}
	}
	log.Printf("server.shutdown.wait timeout=%s", shutdownTimeout)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server.shutdown.warn err=%v (force close)", err)
		_ = srv.Close()
	}

	st.Close()

	remaining := "n/a"
	if dl, ok := shutdownCtx.Deadline(); ok {
		if r := time.Until(dl); r > 0 {
			remaining = r.String()
		} else {
			remaining = "0s"
		}
	}
	log.Printf("server.shutdown.done graceful=%v remaining=%s", shutdownCtx.Err() == nil, remaining)
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

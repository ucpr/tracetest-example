package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	chi "github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/ucpr/tracetest-example/gopkg/trace"
)

const gracefulShutdownTimeout = 3 * time.Second

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	defer stop()

	_, cleanup := trace.NewTracerProvider(ctx, "gateway", "v1.0.0")
	defer cleanup()

	mux := chi.NewRouter()
	mux.Use(otelhttp.NewMiddleware("gateway"))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /hoge", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("GET /hoge", func(w http.ResponseWriter, r *http.Request) {
		client := http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}
		req, err := http.NewRequestWithContext(r.Context(), "GET", "http://service:8081", nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}
		resp, err := client.Do(req) // Service にリクエストを転送
		if err != nil {
			http.Error(w, "Service A unreachable", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response body", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(body)
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	go func() {
		slog.Info("Start server", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	<-ctx.Done()
	tctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(tctx); err != nil {
		panic(err)
	}

	slog.Info("Shutdown")
}

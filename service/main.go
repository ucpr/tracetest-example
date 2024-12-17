package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	chi "github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/uptrace/opentelemetry-go-extra/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/ucpr/tracetest-example/gopkg/trace"
)

const (
	POSTGRES_URL = "postgres://user:password@postgres:5432/dbname?sslmode=disable"

	gracefulShutdownTimeout = 3 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGKILL)
	defer stop()

	_, cleanup := trace.NewTracerProvider(ctx, "service", "v1.0.0")
	defer cleanup()

	db, err := otelsql.Open("postgres", POSTGRES_URL, otelsql.WithAttributes())
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	mux := chi.NewRouter()
	mux.Use(otelhttp.NewMiddleware("service"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var value string
		err := db.QueryRowContext(r.Context(), "SELECT value FROM example WHERE id=1").Scan(&value)
		if err != nil {
			http.Error(w, "Error querying database", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "Value from DB: %s", value)
	})

	srv := &http.Server{
		Addr:    ":8081",
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

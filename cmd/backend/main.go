// Command backend starts an internal CRUD API server instance.
// Each instance identifies itself via the INSTANCE_NAME environment variable,
// which is returned in health-check responses so you can observe round-robin
// load balancing in action.
package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FranAndrews/GateWayr/internal/handler"
	"github.com/FranAndrews/GateWayr/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	instanceName := os.Getenv("INSTANCE_NAME")
	if instanceName == "" {
		instanceName, _ = os.Hostname()
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}
	addr := ":" + port

	// ── Initialise store ────────────────────────────────────────────────
	var appStore store.Store
	var db *sql.DB

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		var err error
		db, err = sql.Open("pgx", dbURL)
		if err != nil {
			logger.Error("failed to open database connection", slog.String("error", err.Error()))
			os.Exit(1)
		}
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		ctxInit, cancelInit := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelInit()

		if err := db.PingContext(ctxInit); err != nil {
			logger.Error("failed to connect to database", slog.String("error", err.Error()))
			os.Exit(1)
		}

		pgStore := store.NewPostgresStore(db)
		if err := pgStore.InitSchema(ctxInit); err != nil {
			logger.Error("failed to initialize schema", slog.String("error", err.Error()))
			os.Exit(1)
		}

		logger.Info("connected to PostgreSQL database", slog.String("instance", instanceName))
		appStore = pgStore
	} else {
		logger.Info("using in-memory store (DATABASE_URL not set)", slog.String("instance", instanceName))
		appStore = store.NewMemoryStore()
	}

	// ── Register routes ─────────────────────────────────────────────────
	mux := http.NewServeMux()

	appHandler := handler.NewApplicationHandler(appStore)
	appHandler.RegisterRoutes(mux)

	mux.HandleFunc("GET /health", handler.HealthHandler())

	// ── Start HTTP server ───────────────────────────────────────────────
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("backend starting",
			slog.String("instance", instanceName),
			slog.String("addr", addr),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down", slog.String("instance", instanceName))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
	}

	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("database close error", slog.String("error", err.Error()))
		}
	}

	logger.Info("backend stopped", slog.String("instance", instanceName))
}

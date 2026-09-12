// Command gateway starts the API Gateway / Reverse Proxy server.
// It loads configuration, initialises the round-robin load balancer,
// wraps the reverse proxy with the middleware chain, and serves HTTP.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FranAndrews/GateWayr/internal/loadbalancer"
	"github.com/FranAndrews/GateWayr/internal/middleware"
	"github.com/FranAndrews/GateWayr/internal/proxy"
	"gopkg.in/yaml.v3"
)

// Config mirrors the structure of config.yaml.
type Config struct {
	Server struct {
		Port         string        `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
		IdleTimeout  time.Duration `yaml:"idle_timeout"`
	} `yaml:"server"`
	Backends []struct {
		URL string `yaml:"url"`
	} `yaml:"backends"`
	Middleware struct {
		Auth struct {
			Enabled   bool     `yaml:"enabled"`
			APIKeys   []string `yaml:"api_keys"`
			SkipPaths []string `yaml:"skip_paths"`
		} `yaml:"auth"`
		RateLimit struct {
			Enabled           bool    `yaml:"enabled"`
			RequestsPerSecond float64 `yaml:"requests_per_second"`
			Burst             int     `yaml:"burst"`
		} `yaml:"rate_limit"`
		Logging struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"logging"`
	} `yaml:"middleware"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// ── Load configuration ──────────────────────────────────────────────
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// ── Initialise load balancer ────────────────────────────────────────
	var backendURLs []string
	for _, b := range cfg.Backends {
		backendURLs = append(backendURLs, b.URL)
	}

	lb, err := loadbalancer.New(backendURLs)
	if err != nil {
		logger.Error("failed to initialise load balancer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("load balancer initialised", slog.Int("backends", len(backendURLs)))

	// ── Build handler chain ─────────────────────────────────────────────
	// The core handler is the reverse proxy.
	var handler http.Handler = proxy.New(lb, logger)

	// Wrap with middleware (applied in reverse order — outermost runs first).
	if cfg.Middleware.Auth.Enabled {
		handler = middleware.APIKeyAuth(cfg.Middleware.Auth.APIKeys, cfg.Middleware.Auth.SkipPaths)(handler)
		logger.Info("middleware enabled: API key authentication")
	}

	if cfg.Middleware.RateLimit.Enabled {
		handler = middleware.RateLimiter(cfg.Middleware.RateLimit.RequestsPerSecond, cfg.Middleware.RateLimit.Burst)(handler)
		logger.Info("middleware enabled: rate limiter",
			slog.Float64("rps", cfg.Middleware.RateLimit.RequestsPerSecond),
			slog.Int("burst", cfg.Middleware.RateLimit.Burst),
		)
	}

	if cfg.Middleware.Logging.Enabled {
		handler = middleware.Logger(logger)(handler)
		logger.Info("middleware enabled: request logging")
	}

	// ── Start HTTP server ───────────────────────────────────────────────
	listenAddr := cfg.Server.Port
	if listenAddr == "" {
		listenAddr = ":8080"
	}
	// Ensure the address has a colon prefix for net/http.
	if listenAddr[0] != ':' {
		listenAddr = ":" + listenAddr
	}

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown on SIGINT / SIGTERM.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("gateway starting", slog.String("addr", listenAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-stop
	logger.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("gateway stopped")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

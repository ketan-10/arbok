package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/knadh/koanf"
	"github.com/mr-karan/arbok/internal/api"
	"github.com/mr-karan/arbok/internal/auth"
	"github.com/mr-karan/arbok/internal/registry"
	"github.com/mr-karan/arbok/internal/tunnel"
)

var (
	// buildString is injected at build time
	buildString = "unknown"
)

func main() {
	// Create main context that cancels on SIGINT/SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Load configuration
	ko := initConfig("config.sample.toml", "ARBOK_SERVER")
	logger := initLogger(ko)

	logger.Info("starting arbok server", slog.String("version", buildString))

	// Parse configuration
	cfg, err := parseConfig(ko)
	if err != nil {
		logger.Error("config error", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize WireGuard tunnel
	tun, err := tunnel.New(tunnel.PeerOpts{
		Logger:     logger,
		Verbose:    cfg.App.Verbose,
		CIDR:       cfg.Server.CIDR,
		ListenPort: cfg.Server.ListenPort,
		PrivateKey: cfg.Server.PrivateKey,
	})
	if err != nil {
		logger.Error("failed to initialize tunnel", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize registry
	reg, err := registry.NewRegistry(ctx, registry.Config{
		CIDR:            cfg.Server.CIDR,
		DefaultTTL:      cfg.Tunnel.DefaultTTL,
		CleanupInterval: cfg.Tunnel.CleanupInterval,
	}, logger)
	if err != nil {
		logger.Error("failed to initialize registry", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize authenticator
	authenticator := auth.New(cfg.Auth.APIKeys, logger)

	// Initialize API server
	// Use endpoint from config, or fallback to domain:port
	endpoint := cfg.Server.Endpoint
	if endpoint == "" {
		endpoint = fmt.Sprintf("%s:%d", cfg.App.Domain, cfg.Server.ListenPort)
	}
	
	apiServer := api.NewAPIServer(api.Config{
		ListenAddr:       cfg.HTTP.ListenAddr,
		Domain:           cfg.App.Domain,
		WireGuardPort:    cfg.Server.ListenPort,
		WireGuardEndpoint: endpoint,
		AllowedOrigins:   cfg.HTTP.AllowedOrigins,
	}, logger, tun, reg, authenticator)

	// Start services
	var wg sync.WaitGroup

	// Start WireGuard tunnel
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tun.Up(ctx); err != nil {
			logger.Error("tunnel error", slog.Any("error", err))
		}
	}()

	// Start HTTP API server
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := apiServer.Start(ctx); err != nil {
			logger.Error("api server error", "error", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info("shutting down")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Close registry (cleans up tunnels)
	if err := reg.Close(); err != nil {
		logger.Error("registry shutdown error", "error", err)
	}

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("shutdown complete")
	case <-shutdownCtx.Done():
		logger.Warn("shutdown timeout exceeded")
	}
}

// Config represents the application configuration
type Config struct {
	App struct {
		Verbose bool   `toml:"verbose"`
		Domain  string `toml:"domain"`
	} `toml:"app"`

	Auth struct {
		APIKeys []string `toml:"api_keys"`
	} `toml:"auth"`

	Tunnel struct {
		DefaultTTL      time.Duration `toml:"default_ttl"`
		CleanupInterval time.Duration `toml:"cleanup_interval"`
	} `toml:"tunnel"`

	Server struct {
		CIDR       string `toml:"cidr"`
		ListenPort int    `toml:"listen_port"`
		PrivateKey string `toml:"private_key"`
		Endpoint   string `toml:"endpoint"`
	} `toml:"server"`

	HTTP struct {
		ListenAddr     string   `toml:"listen_addr"`
		AllowedOrigins []string `toml:"allowed_origins"`
	} `toml:"http"`
}

// parseConfig parses and validates the configuration
func parseConfig(ko *koanf.Koanf) (*Config, error) {
	var cfg Config

	// Set defaults
	cfg.App.Verbose = ko.Bool("app.verbose")
	cfg.App.Domain = ko.String("app.domain")
	
	cfg.Auth.APIKeys = ko.Strings("auth.api_keys")
	
	cfg.Tunnel.DefaultTTL = ko.Duration("tunnel.default_ttl")
	if cfg.Tunnel.DefaultTTL == 0 {
		cfg.Tunnel.DefaultTTL = 24 * time.Hour
	}
	
	cfg.Tunnel.CleanupInterval = ko.Duration("tunnel.cleanup_interval")
	if cfg.Tunnel.CleanupInterval == 0 {
		cfg.Tunnel.CleanupInterval = 5 * time.Minute
	}
	
	cfg.Server.CIDR = ko.String("server.cidr")
	cfg.Server.ListenPort = ko.Int("server.listen_port")
	cfg.Server.PrivateKey = ko.String("server.private_key")
	cfg.Server.Endpoint = ko.String("server.endpoint")
	
	cfg.HTTP.ListenAddr = ko.String("http.listen_addr")
	cfg.HTTP.AllowedOrigins = ko.Strings("http.allowed_origins")

	// Validation
	if cfg.App.Domain == "" {
		return nil, fmt.Errorf("app.domain is required")
	}
	if cfg.Server.CIDR == "" {
		return nil, fmt.Errorf("server.cidr is required")
	}
	if cfg.Server.PrivateKey == "" {
		return nil, fmt.Errorf("server.private_key is required")
	}

	return &cfg, nil
}
package api

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mr-karan/arbok/internal/auth"
	"github.com/mr-karan/arbok/internal/metrics"
	"github.com/mr-karan/arbok/internal/middleware"
	"github.com/mr-karan/arbok/internal/registry"
	"github.com/mr-karan/arbok/internal/tunnel"
	"github.com/zerodha/logf"
)

//go:embed web/*
var webFiles embed.FS

// Server handles HTTP API requests
type Server struct {
	cfg      Config
	logger   logf.Logger
	tun      *tunnel.Tunnel
	registry *registry.Registry
	auth     *auth.Authenticator
	router   *mux.Router
}

// Config holds server configuration
type Config struct {
	ListenAddr      string
	Domain          string
	WireGuardPort   int
	AllowedOrigins  []string
}

// NewServer creates a new API server
func NewAPIServer(cfg Config, logger logf.Logger, tun *tunnel.Tunnel, reg *registry.Registry, auth *auth.Authenticator) *Server {
	s := &Server{
		cfg:      cfg,
		logger:   logger,
		tun:      tun,
		registry: reg,
		auth:     auth,
		router:   mux.NewRouter(),
	}
	
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Global middleware for all routes
	s.router.Use(
		middleware.Recovery(s.logger),
		middleware.Logger(s.logger),
		middleware.CORS(s.cfg.AllowedOrigins),
	)
	
	// Static website
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		s.logger.Error("failed to create web filesystem", "error", err)
	} else {
		s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(webFS))))
		s.router.HandleFunc("/", s.handleWebsite).Methods("GET")
	}
	
	// Health and metrics endpoints
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/metrics", metrics.Handler()).Methods("GET")
	
	// Protected API endpoints
	api := s.router.PathPrefix("/api").Subrouter()
	api.Use(s.auth.Middleware)
	api.HandleFunc("/tunnel/{port:[0-9]+}", s.handleCreateTunnel).Methods("POST")
	api.HandleFunc("/tunnel/{id}", s.handleGetTunnel).Methods("GET")
	api.HandleFunc("/tunnel/{id}", s.handleDeleteTunnel).Methods("DELETE")
	api.HandleFunc("/tunnels", s.handleListTunnels).Methods("GET")
	
	// Simple tunnel provisioning
	s.router.HandleFunc("/{port:[0-9]+}", s.handleProvisionSimple).Methods("GET")
	
	// Tunnel traffic proxy
	s.router.PathPrefix("/").HandlerFunc(s.handleTunnelProxy)
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	server := &http.Server{
		Addr:         s.cfg.ListenAddr,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		s.logger.Info("shutting down http server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("http server shutdown error", "error", err)
		}
	}()
	
	s.logger.Info("starting http server", "addr", s.cfg.ListenAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server error: %w", err)
	}
	
	return nil
}
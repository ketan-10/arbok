package auth

import (
	"context"
	"crypto/subtle"
	"log/slog"
	"net/http"
	"strings"
	
	"github.com/mr-karan/arbok/internal/metrics"
)

// contextKey is a custom type for context keys
type contextKey string

const (
	// ContextKeyAPIKey is the context key for the API key
	ContextKeyAPIKey contextKey = "api_key"
	
	// HeaderAPIKey is the header name for API key
	HeaderAPIKey = "X-API-Key"
	
	// BearerPrefix is the bearer token prefix
	BearerPrefix = "Bearer "
)

// Authenticator handles API authentication
type Authenticator struct {
	keys   map[string]bool
	logger *slog.Logger
}

// New creates a new authenticator
func New(apiKeys []string, logger *slog.Logger) *Authenticator {
	keys := make(map[string]bool, len(apiKeys))
	for _, key := range apiKeys {
		if key != "" {
			keys[key] = true
		}
	}
	
	return &Authenticator{
		keys:   keys,
		logger: logger,
	}
}

// Middleware returns HTTP middleware for authentication
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Skip auth if no keys configured (open mode)
		if len(a.keys) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		
		apiKey := a.extractAPIKey(r)
		if apiKey == "" {
			metrics.AuthFailures.Inc()
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}
		
		if !a.isValidKey(apiKey) {
			metrics.AuthFailures.Inc()
			a.logger.Warn("invalid API key attempt", slog.String("ip", r.RemoteAddr))
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}
		
		metrics.AuthSuccesses.Inc()
		
		// Add API key to context
		ctx := context.WithValue(r.Context(), ContextKeyAPIKey, apiKey)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractAPIKey extracts the API key from the request
func (a *Authenticator) extractAPIKey(r *http.Request) string {
	// Check header first
	if key := r.Header.Get(HeaderAPIKey); key != "" {
		return key
	}
	
	// Check Authorization header with Bearer token
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(auth, BearerPrefix) {
			return strings.TrimPrefix(auth, BearerPrefix)
		}
	}
	
	// Check query parameter as fallback
	return r.URL.Query().Get("api_key")
}

// isValidKey checks if the API key is valid using constant-time comparison
func (a *Authenticator) isValidKey(key string) bool {
	// Use constant-time comparison to prevent timing attacks
	for validKey := range a.keys {
		if subtle.ConstantTimeCompare([]byte(key), []byte(validKey)) == 1 {
			return true
		}
	}
	return false
}

// GetAPIKey retrieves the API key from the request context
func GetAPIKey(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(ContextKeyAPIKey).(string)
	return key, ok
}
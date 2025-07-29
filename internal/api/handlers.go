package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/mr-karan/arbok/internal/tunnel"
)

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// TunnelResponse represents a tunnel in API responses
type TunnelResponse struct {
	ID        string    `json:"id"`
	Subdomain string    `json:"subdomain"`
	URL       string    `json:"url"`
	Port      uint16    `json:"port"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	TTL       string    `json:"ttl"`
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log error but don't write again to avoid superfluous write
		_ = err
	}
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

// handleCreateTunnel handles tunnel creation requests
func (s *Server) handleCreateTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	port, err := strconv.ParseUint(vars["port"], 10, 16)
	if err != nil || port == 0 || port > 65535 {
		writeError(w, http.StatusBadRequest, "INVALID_PORT", "Invalid port number")
		return
	}
	
	// Create tunnel
	t, err := s.registry.CreateTunnel(uint16(port))
	if err != nil {
		s.logger.Error("failed to create tunnel", "error", err, "port", port)
		writeError(w, http.StatusInternalServerError, "TUNNEL_CREATE_FAILED", "Failed to create tunnel")
		return
	}
	
	// Add peer to WireGuard
	if err := s.tun.AddPeer(t.PublicKey, t.AllowedIP); err != nil {
		s.logger.Error("failed to add peer", "error", err, "tunnel_id", t.ID)
		_ = s.registry.DeleteTunnel(t.ID)
		writeError(w, http.StatusInternalServerError, "PEER_ADD_FAILED", "Failed to configure tunnel")
		return
	}
	
	// Return tunnel info
	resp := TunnelResponse{
		ID:        t.ID,
		Subdomain: t.Subdomain,
		URL:       fmt.Sprintf("https://%s.%s", t.Subdomain, s.cfg.Domain),
		Port:      t.Port,
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
		TTL:       t.TTL().String(),
	}
	
	writeJSON(w, http.StatusCreated, resp)
}

// handleGetTunnel handles tunnel info requests
func (s *Server) handleGetTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]
	
	t := s.registry.GetTunnel(tunnelID)
	if t == nil {
		writeError(w, http.StatusNotFound, "TUNNEL_NOT_FOUND", "Tunnel not found")
		return
	}
	
	resp := TunnelResponse{
		ID:        t.ID,
		Subdomain: t.Subdomain,
		URL:       fmt.Sprintf("https://%s.%s", t.Subdomain, s.cfg.Domain),
		Port:      t.Port,
		CreatedAt: t.CreatedAt,
		ExpiresAt: t.ExpiresAt,
		TTL:       t.TTL().String(),
	}
	
	writeJSON(w, http.StatusOK, resp)
}

// handleDeleteTunnel handles tunnel deletion requests
func (s *Server) handleDeleteTunnel(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tunnelID := vars["id"]
	
	t := s.registry.GetTunnel(tunnelID)
	if t == nil {
		writeError(w, http.StatusNotFound, "TUNNEL_NOT_FOUND", "Tunnel not found")
		return
	}
	
	// Remove peer from WireGuard
	if err := s.tun.RemovePeer(t.PublicKey); err != nil {
		s.logger.Error("failed to remove peer", "error", err, "tunnel_id", t.ID)
	}
	
	// Delete from registry
	if err := s.registry.DeleteTunnel(tunnelID); err != nil {
		s.logger.Error("failed to delete tunnel", "error", err, "tunnel_id", tunnelID)
		writeError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete tunnel")
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// handleListTunnels handles tunnel listing requests
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	tunnels := s.registry.ListTunnels()
	
	resp := make([]TunnelResponse, 0, len(tunnels))
	for _, t := range tunnels {
		resp = append(resp, TunnelResponse{
			ID:        t.ID,
			Subdomain: t.Subdomain,
			URL:       fmt.Sprintf("https://%s.%s", t.Subdomain, s.cfg.Domain),
			Port:      t.Port,
			CreatedAt: t.CreatedAt,
			ExpiresAt: t.ExpiresAt,
			TTL:       t.TTL().String(),
		})
	}
	
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tunnels": resp,
		"count":   len(resp),
	})
}

// handleProvisionSimple handles simple tunnel provisioning (curl-friendly)
func (s *Server) handleProvisionSimple(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	port, err := strconv.ParseUint(vars["port"], 10, 16)
	if err != nil || port == 0 || port > 65535 {
		http.Error(w, "Invalid port number", http.StatusBadRequest)
		return
	}
	
	// Create tunnel
	t, err := s.registry.CreateTunnel(uint16(port))
	if err != nil {
		s.logger.Error("failed to create tunnel", "error", err, "port", port)
		http.Error(w, "Failed to create tunnel", http.StatusInternalServerError)
		return
	}
	
	// Add peer to WireGuard
	if err := s.tun.AddPeer(t.PublicKey, t.AllowedIP); err != nil {
		s.logger.Error("failed to add peer", "error", err, "tunnel_id", t.ID)
		_ = s.registry.DeleteTunnel(t.ID)
		http.Error(w, "Failed to configure tunnel", http.StatusInternalServerError)
		return
	}
	
	// Generate WireGuard config
	config := s.generateWireGuardConfig(t)
	
	// Add helpful instructions
	instructions := fmt.Sprintf(`# Arbok Tunnel Configuration
# Generated: %s
# Expires: %s (in %s)
#
# Your local service on port %d is now accessible at:
# https://%s.%s
#
# Usage:
#   1. Save this config: curl %s/%d > wg.conf
#   2. Start tunnel: sudo wg-quick up ./wg.conf  
#   3. Stop tunnel: sudo wg-quick down ./wg.conf
#
%s`, 
		t.CreatedAt.Format(time.RFC3339),
		t.ExpiresAt.Format(time.RFC3339),
		t.TTL().Round(time.Minute),
		t.Port, 
		t.Subdomain, 
		s.cfg.Domain,
		s.cfg.Domain,
		t.Port,
		s.cfg.Domain,
		t.Port,
		config,
	)
	
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.conf"`, t.Subdomain))
	fmt.Fprint(w, instructions)
}


// generateWireGuardConfig generates a WireGuard configuration
func (s *Server) generateWireGuardConfig(t *tunnel.Info) string {
	serverEndpoint := s.cfg.WireGuardEndpoint
	
	tunnelURL := fmt.Sprintf("https://%s.%s", t.Subdomain, s.cfg.Domain)
	
	return fmt.Sprintf(`[Interface]
Address = %s/32
PrivateKey = %s
PostUp = echo "🐍 Arbok tunnel active! Local port %d → %s"

[Peer]
PublicKey = %s
AllowedIPs = 10.100.0.1/32
Endpoint = %s
PersistentKeepalive = 25`, 
		t.AllowedIP,
		t.PrivateKey,
		t.Port,
		tunnelURL,
		s.tun.GetPublicKey(), 
		serverEndpoint,
	)
}

// handleTunnelProxy proxies traffic to tunnels
func (s *Server) handleTunnelProxy(w http.ResponseWriter, r *http.Request) {
	// Extract subdomain
	host := r.Host
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}
	
	parts := strings.Split(host, ".")
	if len(parts) < 2 {
		s.logger.Debug("tunnel proxy: invalid host", "host", host, "parts", len(parts))
		writeError(w, http.StatusBadRequest, "INVALID_HOST", "Invalid host header")
		return
	}
	
	subdomain := parts[0]
	s.logger.Debug("tunnel proxy: looking for tunnel", "host", host, "subdomain", subdomain)
	t := s.registry.GetTunnelBySubdomain(subdomain)
	if t == nil {
		s.logger.Debug("tunnel proxy: tunnel not found", "subdomain", subdomain)
		writeError(w, http.StatusNotFound, "TUNNEL_NOT_FOUND", "Tunnel not found")
		return
	}
	
	s.logger.Debug("tunnel proxy: found tunnel", "subdomain", subdomain, "tunnel_id", t.ID)
	
	// Update traffic stats
	defer func() {
		// This is a simplified version - in production you'd track actual bytes
		s.registry.UpdateTraffic(t.ID, 0, 0)
	}()
	
	// Use the proxy handler
	s.handleTunnelTrafficWithProxy(w, r)
}

// handleWebsite serves the embedded website
func (s *Server) handleWebsite(w http.ResponseWriter, r *http.Request) {
	content, err := webFiles.ReadFile("web/index.html")
	if err != nil {
		s.logger.Error("failed to read website", "error", err)
		http.Error(w, "Website unavailable", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(content); err != nil {
		s.logger.Error("failed to write website response", "error", err)
	}
}

// handleClientScript serves the arbok client helper script
func (s *Server) handleClientScript(w http.ResponseWriter, r *http.Request) {
	script := `#!/bin/bash
# Arbok Client - One command tunnel management
# Usage: curl -O https://server/client && chmod +x client && ./client start 3000

ARBOK_SERVER="${ARBOK_SERVER:-` + s.cfg.Domain + `}"
# ... (rest of the client script would be embedded here)
`
	
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=\"arbok\"")
	fmt.Fprint(w, script)
}
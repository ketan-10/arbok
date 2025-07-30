package api

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// createReverseProxy creates a reverse proxy for a tunnel using netstack
func (s *Server) createReverseProxy(targetIP string, port uint16) *httputil.ReverseProxy {
	target := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", targetIP, port),
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	
	// Get netstack from tunnel for userspace networking
	tnet := s.tun.GetNetstack()
	
	// Customize the transport to use netstack (userspace WireGuard networking)
	proxy.Transport = &http.Transport{
		DialContext:           tnet.DialContext, // Use netstack instead of kernel networking
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Customize error handling
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.logger.Error("proxy error", "error", err, "target", target.String())
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Modify request headers
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host

		// Add X-Forwarded headers
		if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
			if prior, ok := req.Header["X-Forwarded-For"]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
			req.Header.Set("X-Forwarded-For", clientIP)
		}
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Forwarded-Proto", "https")
		
		// Remove hop-by-hop headers
		for _, h := range hopHeaders {
			req.Header.Del(h)
		}
	}

	// Modify response headers
	proxy.ModifyResponse = func(resp *http.Response) error {
		// Remove hop-by-hop headers from response
		for _, h := range hopHeaders {
			resp.Header.Del(h)
		}
		return nil
	}

	return proxy
}

// extractSubdomain extracts the subdomain from a host header value.
// It handles port stripping and returns just the subdomain portion.
func extractSubdomain(host string) string {
	// Remove port if present
	if idx := strings.IndexByte(host, ':'); idx != -1 {
		host = host[:idx]
	}
	
	// Extract subdomain (first part before first dot)
	if idx := strings.IndexByte(host, '.'); idx != -1 {
		return host[:idx]
	}
	return host
}

// handleTunnelTrafficWithProxy handles incoming traffic and proxies it to the tunnel
func (s *Server) handleTunnelTrafficWithProxy(w http.ResponseWriter, r *http.Request) {
	// Extract subdomain from host
	subdomain := extractSubdomain(r.Host)
	if subdomain == "" {
		http.Error(w, "Invalid host header", http.StatusBadRequest)
		return
	}
	tunnel := s.registry.GetTunnelBySubdomain(subdomain)
	if tunnel == nil {
		http.Error(w, "Tunnel not found", http.StatusNotFound)
		return
	}

	// Handle WebSocket upgrade
	if isWebSocketRequest(r) {
		s.handleWebSocket(w, r, tunnel.AllowedIP, tunnel.Port)
		return
	}

	// Create and use reverse proxy
	proxy := s.createReverseProxy(tunnel.AllowedIP, tunnel.Port)
	proxy.ServeHTTP(w, r)
}

// isWebSocketRequest checks if the request is a WebSocket upgrade request
func isWebSocketRequest(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request, targetIP string, port uint16) {
	// Dial the backend WebSocket server
	targetURL := fmt.Sprintf("ws://%s:%d%s", targetIP, port, r.URL.Path)
	if r.URL.RawQuery != "" {
		targetURL += "?" + r.URL.RawQuery
	}

	targetConn, resp, err := s.websocketDial(targetURL, r.Header)
	if err != nil {
		s.logger.Error("websocket dial error", "error", err, "target", targetURL)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	// Hijack the client connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		s.logger.Error("hijack error", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Write the WebSocket upgrade response
	if err := writeWebSocketResponse(clientConn, resp); err != nil {
		s.logger.Error("write response error", "error", err)
		return
	}

	// Use context for proper cancellation
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Proxy data between connections with proper cleanup
	errc := make(chan error, 2)
	go func() {
		defer cancel() // Cancel context when one direction completes
		_, err := io.Copy(targetConn, clientConn)
		errc <- err
	}()
	go func() {
		defer cancel() // Cancel context when one direction completes
		_, err := io.Copy(clientConn, targetConn)
		errc <- err
	}()

	// Wait for either copy to complete or context cancellation
	select {
	case <-ctx.Done():
		return
	case <-errc:
		return
	}
}

// websocketDial dials a WebSocket connection using the tunnel's netstack
func (s *Server) websocketDial(targetURL string, headers http.Header) (net.Conn, *http.Response, error) {
	// Parse the URL
	u, err := url.Parse(targetURL)
	if err != nil {
		return nil, nil, err
	}

	// Get netstack from tunnel for userspace networking
	tnet := s.tun.GetNetstack()
	
	// Dial TCP connection using netstack (userspace WireGuard networking)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := tnet.DialContext(ctx, "tcp", u.Host)
	if err != nil {
		return nil, nil, err
	}

	// Send WebSocket upgrade request
	req := &http.Request{
		Method: "GET",
		URL:    u,
		Header: make(http.Header),
		Host:   u.Host,
	}

	// Copy relevant headers
	for k, v := range headers {
		if k == "Host" || k == "Upgrade" || k == "Connection" || k == "Sec-Websocket-Key" ||
			k == "Sec-Websocket-Version" || strings.HasPrefix(k, "Sec-Websocket-") {
			req.Header[k] = v
		}
	}

	if err := req.Write(conn); err != nil {
		conn.Close()
		return nil, nil, err
	}

	// Read response
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	return conn, resp, nil
}

// writeWebSocketResponse writes a WebSocket upgrade response
func writeWebSocketResponse(conn net.Conn, resp *http.Response) error {
	// Write status line
	if _, err := fmt.Fprintf(conn, "HTTP/1.1 %d %s\r\n", resp.StatusCode, resp.Status); err != nil {
		return err
	}

	// Write headers
	for k, values := range resp.Header {
		for _, v := range values {
			if _, err := fmt.Fprintf(conn, "%s: %s\r\n", k, v); err != nil {
				return err
			}
		}
	}

	// End headers
	if _, err := fmt.Fprintf(conn, "\r\n"); err != nil {
		return err
	}

	return nil
}

// Hop-by-hop headers that should be removed
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}
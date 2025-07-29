package metrics

import (
	"fmt"
	"net/http"

	"github.com/VictoriaMetrics/metrics"
)

var (
	// Tunnel metrics
	TunnelsActive   = metrics.NewGauge(`arbok_tunnels_active`, nil)
	TunnelsCreated  = metrics.NewCounter(`arbok_tunnels_created_total`)
	TunnelsDeleted  = metrics.NewCounter(`arbok_tunnels_deleted_total`)
	TunnelsExpired  = metrics.NewCounter(`arbok_tunnels_expired_total`)
	
	// HTTP metrics
	HTTPRequestsTotal = metrics.NewCounter(`arbok_http_requests_total`)
	HTTPRequestDuration = metrics.NewHistogram(`arbok_http_request_duration_seconds`)
	HTTPBytesProxied = metrics.NewCounter(`arbok_http_bytes_proxied_total`)
	
	// WireGuard metrics
	WireGuardPeersActive = metrics.NewGauge(`arbok_wireguard_peers_active`, nil)
	WireGuardErrors = metrics.NewCounter(`arbok_wireguard_errors_total`)
	
	// IP pool metrics
	IPPoolAvailable = metrics.NewGauge(`arbok_ip_pool_available`, nil)
	IPPoolExhausted = metrics.NewCounter(`arbok_ip_pool_exhausted_total`)
	
	// Auth metrics
	AuthFailures = metrics.NewCounter(`arbok_auth_failures_total`)
	AuthSuccesses = metrics.NewCounter(`arbok_auth_successes_total`)
)

// Handler returns the metrics handler for Prometheus scraping
func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	}
}

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, path string, statusCode int, duration float64) {
	HTTPRequestsTotal.Inc()
	HTTPRequestDuration.Update(duration)
	
	// You can also use labeled metrics if needed
	counter := metrics.GetOrCreateCounter(
		fmt.Sprintf(`arbok_http_requests_total{method=%q,path=%q,status="%d"}`, 
			method, path, statusCode))
	counter.Inc()
}
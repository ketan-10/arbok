package registry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mr-karan/arbok/internal/metrics"
	"github.com/mr-karan/arbok/internal/tunnel"
	"github.com/zerodha/logf"
)

// Config holds registry configuration
type Config struct {
	CIDR           string
	DefaultTTL     time.Duration
	CleanupInterval time.Duration
}

// Registry manages active tunnels
type Registry struct {
	cfg    Config
	logger logf.Logger
	
	mu          sync.RWMutex
	tunnels     map[string]*tunnel.Info
	bySubdomain map[string]*tunnel.Info
	
	ipPool   *IPPool
	keyGen   KeyGenerator
	nameGen  NameGenerator
	
	cleanupTimer *time.Timer
	ctx          context.Context
	cancel       context.CancelFunc
}

// New creates a new registry
func NewRegistry(ctx context.Context, cfg Config, logger logf.Logger) (*Registry, error) {
	pool, err := NewIPPool(cfg.CIDR)
	if err != nil {
		return nil, fmt.Errorf("failed to create IP pool: %w", err)
	}
	
	ctx, cancel := context.WithCancel(ctx)
	
	r := &Registry{
		cfg:         cfg,
		logger:      logger,
		tunnels:     make(map[string]*tunnel.Info),
		bySubdomain: make(map[string]*tunnel.Info),
		ipPool:      pool,
		keyGen:      &WireGuardKeyGenerator{},
		nameGen:     &FriendlyNameGenerator{},
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// Start cleanup routine
	go r.cleanupRoutine()
	
	// Update metrics
	metrics.IPPoolAvailable.Set(float64(pool.Available()))
	
	return r, nil
}

// CreateTunnel creates a new tunnel
func (r *Registry) CreateTunnel(port uint16) (*tunnel.Info, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Allocate IP
	ip, err := r.ipPool.Allocate()
	if err != nil {
		metrics.IPPoolExhausted.Inc()
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}
	
	// Generate keys
	privateKey, publicKey, err := r.keyGen.Generate()
	if err != nil {
		r.ipPool.Release(ip)
		return nil, fmt.Errorf("failed to generate keys: %w", err)
	}
	
	// Generate subdomain
	subdomain := r.nameGen.Generate()
	
	// Create tunnel
	t := &tunnel.Info{
		ID:         uuid.New().String(),
		Subdomain:  subdomain,
		Port:       port,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		AllowedIP:  ip.String(),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(r.cfg.DefaultTTL),
		LastSeen:   time.Now(),
	}
	
	r.tunnels[t.ID] = t
	r.bySubdomain[t.Subdomain] = t
	
	// Update metrics
	metrics.TunnelsActive.Inc()
	metrics.TunnelsCreated.Inc()
	metrics.IPPoolAvailable.Set(float64(r.ipPool.Available()))
	
	r.logger.Info("tunnel created", 
		"id", t.ID, 
		"subdomain", t.Subdomain,
		"ip", t.AllowedIP,
		"ttl", r.cfg.DefaultTTL)
	
	return t, nil
}

// GetTunnel retrieves a tunnel by ID
func (r *Registry) GetTunnel(id string) *tunnel.Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	t := r.tunnels[id]
	if t != nil {
		t.UpdateLastSeen()
	}
	return t
}

// GetTunnelBySubdomain retrieves a tunnel by subdomain
func (r *Registry) GetTunnelBySubdomain(subdomain string) *tunnel.Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	t := r.bySubdomain[subdomain]
	if t != nil {
		t.UpdateLastSeen()
	}
	return t
}

// DeleteTunnel removes a tunnel
func (r *Registry) DeleteTunnel(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	t, exists := r.tunnels[id]
	if !exists {
		return fmt.Errorf("tunnel not found: %s", id)
	}
	
	return r.deleteTunnelLocked(t)
}

// deleteTunnelLocked removes a tunnel (must be called with lock held)
func (r *Registry) deleteTunnelLocked(t *tunnel.Info) error {
	// Release IP
	if err := r.ipPool.ReleaseString(t.AllowedIP); err != nil {
		r.logger.Error("failed to release IP", "error", err, "ip", t.AllowedIP)
	}
	
	delete(r.tunnels, t.ID)
	delete(r.bySubdomain, t.Subdomain)
	
	// Update metrics
	metrics.TunnelsActive.Dec()
	metrics.TunnelsDeleted.Inc()
	metrics.IPPoolAvailable.Set(float64(r.ipPool.Available()))
	
	r.logger.Info("tunnel deleted", "id", t.ID, "subdomain", t.Subdomain)
	
	return nil
}

// ListTunnels returns all active tunnels
func (r *Registry) ListTunnels() []*tunnel.Info {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tunnels := make([]*tunnel.Info, 0, len(r.tunnels))
	for _, t := range r.tunnels {
		tunnels = append(tunnels, t)
	}
	return tunnels
}

// cleanupRoutine periodically removes expired tunnels
func (r *Registry) cleanupRoutine() {
	ticker := time.NewTicker(r.cfg.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.cleanupExpired()
		}
	}
}

// cleanupExpired removes expired tunnels
func (r *Registry) cleanupExpired() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var expired []*tunnel.Info
	
	for _, t := range r.tunnels {
		if t.IsExpired() {
			expired = append(expired, t)
		}
	}
	
	for _, t := range expired {
		if err := r.deleteTunnelLocked(t); err != nil {
			r.logger.Error("failed to delete expired tunnel", "error", err, "id", t.ID)
		} else {
			metrics.TunnelsExpired.Inc()
		}
	}
	
	if len(expired) > 0 {
		r.logger.Info("cleaned up expired tunnels", "count", len(expired))
	}
}

// Close gracefully shuts down the registry
func (r *Registry) Close() error {
	r.cancel()
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Clean up all tunnels
	for _, t := range r.tunnels {
		r.deleteTunnelLocked(t)
	}
	
	return nil
}

// UpdateTraffic updates traffic statistics for a tunnel
func (r *Registry) UpdateTraffic(id string, bytesIn, bytesOut uint64) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if t, exists := r.tunnels[id]; exists {
		t.BytesIn += bytesIn
		t.BytesOut += bytesOut
		metrics.HTTPBytesProxied.Add(int(bytesIn + bytesOut))
	}
}
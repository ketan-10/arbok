package tunnel

import (
	"time"
)

// Info represents a tunnel connection
type Info struct {
	ID         string    `json:"id"`
	Subdomain  string    `json:"subdomain"`
	Port       uint16    `json:"port"`
	PublicKey  string    `json:"public_key"`
	PrivateKey string    `json:"-"` // Never expose in JSON
	AllowedIP  string    `json:"allowed_ip"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastSeen   time.Time `json:"last_seen"`
	BytesIn    uint64    `json:"bytes_in"`
	BytesOut   uint64    `json:"bytes_out"`
}

// IsExpired checks if the tunnel has expired
func (t *Info) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// UpdateLastSeen updates the last seen timestamp
func (t *Info) UpdateLastSeen() {
	t.LastSeen = time.Now()
}

// TTL returns the time until expiration
func (t *Info) TTL() time.Duration {
	return time.Until(t.ExpiresAt)
}
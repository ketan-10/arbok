package registry

import (
	"fmt"
	"net"
	"sync"
)

// IPPool manages IP address allocation
type IPPool struct {
	mu        sync.Mutex
	network   *net.IPNet
	allocated map[string]bool
	available int
}

// NewIPPool creates a new IP pool from a CIDR
func NewIPPool(cidr string) (*IPPool, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}
	
	// Calculate total available IPs (excluding network and broadcast)
	ones, bits := network.Mask.Size()
	total := 1 << (bits - ones)
	if total > 2 {
		total -= 2 // Remove network and broadcast addresses
	}
	
	return &IPPool{
		network:   network,
		allocated: make(map[string]bool),
		available: total - 1, // -1 for server (.1)
	}, nil
}

// Allocate assigns an available IP address
func (p *IPPool) Allocate() (net.IP, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.available <= 0 {
		return nil, fmt.Errorf("IP pool exhausted")
	}
	
	// Start from .2 (reserve .1 for server)
	ip := make(net.IP, len(p.network.IP))
	copy(ip, p.network.IP)
	
	// Find next available IP
	for i := 2; i < 256; i++ { // Simple implementation for /24
		ip[len(ip)-1] = byte(i)
		
		if !p.network.Contains(ip) {
			break
		}
		
		ipStr := ip.String()
		if !p.allocated[ipStr] {
			p.allocated[ipStr] = true
			p.available--
			return ip, nil
		}
	}
	
	return nil, fmt.Errorf("no available IPs in pool")
}

// Release returns an IP to the pool
func (p *IPPool) Release(ip net.IP) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	ipStr := ip.String()
	if p.allocated[ipStr] {
		delete(p.allocated, ipStr)
		p.available++
		return nil
	}
	
	return fmt.Errorf("IP %s was not allocated", ipStr)
}

// ReleaseString is a convenience method for releasing by string
func (p *IPPool) ReleaseString(ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP: %s", ipStr)
	}
	return p.Release(ip)
}

// Available returns the number of available IPs
func (p *IPPool) Available() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.available
}

// Allocated returns the number of allocated IPs
func (p *IPPool) Allocated() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.allocated)
}
// Package tunnel provides userspace WireGuard interface management
// for secure peer-to-peer networking. It supports dynamic peer addition/removal
// and integrates with netstack for userspace networking operations.
//
// The package implements a WireGuard tunnel that runs entirely in userspace,
// requiring no root privileges or kernel module modifications. It uses
// gvisor's netstack for TCP/IP operations and supports:
//
//   - Dynamic peer configuration via IPC
//   - Automatic IP address management
//   - DNS resolution through configurable servers  
//   - Graceful shutdown and resource cleanup
//
// Example usage:
//
//	opts := PeerOpts{
//		CIDR:       "10.100.0.0/24",
//		ListenPort: 54321,
//		PrivateKey: "base64-encoded-private-key",
//		Logger:     logger,
//	}
//	
//	tunnel, err := New(opts)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer tunnel.Close()
//	
//	// Add a peer
//	err = tunnel.AddPeer("peer-public-key", "10.100.0.2")
package tunnel

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sync"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

const (
	DefaultListenPort = 54321           // Default UDP port for WireGuard
	DefaultCIDR       = "10.100.0.0/24" // Default CIDR for server interface
	DefaultMTU        = 1420            // Default MTU for WireGuard interface
)

// PeerOpts represents configuration options for WireGuard peer initialization.
type PeerOpts struct {
	CIDR       string       // Network CIDR for the tunnel
	ListenPort int          // UDP port for WireGuard to listen on
	PrivateKey string       // Base64-encoded private key
	DNSServers []string     // DNS servers for netstack (optional)
	Verbose    bool         // Enable verbose logging
	Logger     *slog.Logger // Logger instance
}

// Tunnel represents a WireGuard userspace tunnel interface.
type Tunnel struct {
	// Configuration
	logger     *slog.Logger
	privateKey string
	publicKey  string
	listenPort int
	serverIP   netip.Addr
	cidr       string
	
	// WireGuard components
	device *device.Device
	tun    tun.Device
	tnet   *netstack.Net
	
	// Synchronization
	closeMutex sync.Mutex
	closed     bool
}

// validateCIDR validates that the provided CIDR is valid.
func validateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR format: %w", err)
	}
	return nil
}

// getDNSAddrs converts DNS server strings to netip.Addr slice with defaults.
func getDNSAddrs(dnsServers []string) []netip.Addr {
	if len(dnsServers) == 0 {
		// Use Google DNS as default
		return []netip.Addr{
			netip.MustParseAddr("8.8.8.8"),
			netip.MustParseAddr("8.8.4.4"),
		}
	}
	
	addrs := make([]netip.Addr, 0, len(dnsServers))
	for _, dns := range dnsServers {
		if addr, err := netip.ParseAddr(dns); err == nil {
			addrs = append(addrs, addr)
		}
	}
	
	// Fallback to default if no valid DNS servers provided
	if len(addrs) == 0 {
		return []netip.Addr{
			netip.MustParseAddr("8.8.8.8"),
			netip.MustParseAddr("8.8.4.4"),
		}
	}
	
	return addrs
}

// truncateKey safely truncates a key for logging purposes.
func truncateKey(key string) string {
	if len(key) <= 12 {
		return key
	}
	return key[:8] + "..."
}

// New creates a new WireGuard tunnel interface with the specified configuration.
// It initializes a userspace WireGuard device using netstack and configures
// the server's private key and listen port.
//
// The function performs the following operations:
// 1. Validates and sets default configuration values
// 2. Calculates the server IP from the provided CIDR
// 3. Creates a netstack TUN interface
// 4. Configures the WireGuard device with keys and network settings
// 5. Brings the interface up and ready for peer connections
//
// Returns an error if any step fails, with proper resource cleanup.
func New(opts PeerOpts) (*Tunnel, error) {
	// Validate required parameters
	if opts.PrivateKey == "" {
		return nil, fmt.Errorf("private key is required")
	}
	
	// Set default options
	if opts.CIDR == "" {
		opts.CIDR = DefaultCIDR
	}
	if opts.ListenPort == 0 {
		opts.ListenPort = DefaultListenPort
	}
	
	// Validate CIDR
	if err := validateCIDR(opts.CIDR); err != nil {
		return nil, fmt.Errorf("invalid CIDR: %w", err)
	}

	// Parse the CIDR to get the network range
	_, cidrNet, err := net.ParseCIDR(opts.CIDR)
	if err != nil {
		return nil, fmt.Errorf("error parsing CIDR: %w", err)
	}

	// Calculate the server IP (first usable IP in range)
	serverIP := make(net.IP, len(cidrNet.IP))
	copy(serverIP, cidrNet.IP)
	serverIP[len(serverIP)-1] += 1 // .1

	// Convert to netip.Addr
	serverAddr, ok := netip.AddrFromSlice(serverIP)
	if !ok {
		return nil, fmt.Errorf("invalid server IP: %s", serverIP.String())
	}

	// Calculate public key from private key
	pubKey, err := privateKeyToPublicKey(opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error calculating public key: %w", err)
	}

	// Get DNS servers (use defaults if not specified)
	dnsAddrs := getDNSAddrs(opts.DNSServers)
	
	// Create netstack TUN device
	tun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{serverAddr},
		dnsAddrs,
		DefaultMTU,
	)
	if err != nil {
		return nil, fmt.Errorf("error creating netstack TUN: %w", err)
	}

	// Create WireGuard device
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, ""))

	// Convert base64 private key to hex for WireGuard IPC
	privateKeyHex, err := encodeBase64ToHex(opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error converting private key to hex: %w", err)
	}

	// Configure the device with private key and listen port
	config := fmt.Sprintf("private_key=%s\nlisten_port=%d\n", privateKeyHex, opts.ListenPort)
	if err := dev.IpcSet(config); err != nil {
		tun.Close() // Cleanup TUN interface on failure
		return nil, fmt.Errorf("error configuring WireGuard device: %w", err)
	}

	// Bring the device up
	if err := dev.Up(); err != nil {
		dev.Close() // Cleanup device on failure
		tun.Close() // Cleanup TUN interface on failure
		return nil, fmt.Errorf("error bringing WireGuard device up: %w", err)
	}

	return &Tunnel{
		logger:     opts.Logger,
		privateKey: opts.PrivateKey,
		publicKey:  pubKey,
		listenPort: opts.ListenPort,
		serverIP:   serverAddr,
		cidr:       opts.CIDR,
		device:     dev,
		tun:        tun,
		tnet:       tnet,
	}, nil
}

// Up waits for the context to be cancelled and then shuts down the interface
func (tun *Tunnel) Up(ctx context.Context) error {
	// Wait for context cancellation
	<-ctx.Done()

	// Safely cleanup resources
	tun.Close()
	return nil
}

// Close safely shuts down the tunnel resources
func (tun *Tunnel) Close() error {
	tun.closeMutex.Lock()
	defer tun.closeMutex.Unlock()
	
	if tun.closed {
		return nil
	}
	
	// Close device - this automatically closes the associated TUN
	if tun.device != nil {
		tun.device.Close()
		tun.device = nil
	}
	
	// Don't explicitly close tun as device.Close() handles it
	// Setting to nil to prevent double-close attempts
	tun.tun = nil
	
	tun.closed = true
	return nil
}

// GetPublicKey returns the server's public key
func (tun *Tunnel) GetPublicKey() string {
	return tun.publicKey
}

// GetServerIP returns the server's IP address by calculating it from the CIDR
func GetServerIP(cidr string) (string, error) {
	_, cidrNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("error parsing CIDR: %w", err)
	}

	// Calculate the first usable IP in the range (network + 1)
	serverIP := make(net.IP, len(cidrNet.IP))
	copy(serverIP, cidrNet.IP)
	serverIP[len(serverIP)-1] += 1 // Increment last octet to get .1

	return serverIP.String(), nil
}

// GetNetstack returns the netstack network interface for dialing
func (tun *Tunnel) GetNetstack() *netstack.Net {
	return tun.tnet
}

// AddPeer adds a new peer to the userspace WireGuard interface.
// It validates the input parameters and configures the peer with the specified
// public key and allowed IP address.
func (tun *Tunnel) AddPeer(publicKey, allowedIP string) error {
	// Validate input parameters
	if publicKey == "" {
		return fmt.Errorf("public key cannot be empty")
	}
	if net.ParseIP(allowedIP) == nil {
		return fmt.Errorf("invalid IP address: %s", allowedIP)
	}
	
	// Convert base64 public key to hex for WireGuard IPC
	publicKeyHex, err := encodeBase64ToHex(publicKey)
	if err != nil {
		return fmt.Errorf("error converting public key to hex: %w", err)
	}

	// Configure peer using IPC
	config := fmt.Sprintf("public_key=%s\nallowed_ip=%s/32\npersistent_keepalive_interval=25\n",
		publicKeyHex, allowedIP)

	if err := tun.device.IpcSet(config); err != nil {
		return fmt.Errorf("error adding peer to WireGuard: %w", err)
	}

	tun.logger.Info("added peer", 
		slog.String("public_key", truncateKey(publicKey)), 
		slog.String("allowed_ip", allowedIP))
	return nil
}

// RemovePeer removes a peer from the userspace WireGuard interface.
// It validates the public key and removes the peer configuration.
func (tun *Tunnel) RemovePeer(publicKey, allowedIP string) error {
	// Validate input parameters
	if publicKey == "" {
		return fmt.Errorf("public key cannot be empty")
	}
	
	// Convert base64 public key to hex for WireGuard IPC
	publicKeyHex, err := encodeBase64ToHex(publicKey)
	if err != nil {
		return fmt.Errorf("error converting public key to hex: %w", err)
	}

	// Remove peer using IPC
	config := fmt.Sprintf("public_key=%s\nremove=true\n", publicKeyHex)

	if err := tun.device.IpcSet(config); err != nil {
		return fmt.Errorf("error removing peer from WireGuard: %w", err)
	}

	tun.logger.Info("removed peer", 
		slog.String("public_key", truncateKey(publicKey)), 
		slog.String("allowed_ip", allowedIP))
	return nil
}
package tunnel

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"github.com/zerodha/logf"
	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

const (
	WG_INTERFACE          = "wg0"           // Default wireguard interface to create.
	WG_SERVER_LISTEN_PORT = 54321           // Default UDP port to listen.
	WG_SERVER_CIDR        = "10.100.0.0/24" // Default CIDR to use for the server wg interface.
)

// PeerOpts represent options to configure a Wireguard peer.
type PeerOpts struct {
	CIDR       string
	ListenPort int
	PrivateKey string
	Verbose    bool
	Logger     logf.Logger
}

type Tunnel struct {
	dev        *device.Device
	logger     logf.Logger
	privateKey string
	publicKey  string
}

// New initialises a wireguard peer with the give config.
// It starts the wireguard interface. A peer needs to be added separately.
func New(opts PeerOpts) (*Tunnel, error) {
	// Set default options.
	if opts.CIDR == "" {
		opts.CIDR = WG_SERVER_CIDR
	}
	if opts.ListenPort == 0 {
		opts.ListenPort = WG_SERVER_LISTEN_PORT
	}

	// Create an interface for wireguard.
	t, err := tun.CreateTUN(WG_INTERFACE, device.DefaultMTU)
	if err != nil {
		return nil, fmt.Errorf("error creating wg interface: %w", err)
	}

	// Get the link.
	link, err := netlink.LinkByName("wg0")
	if err != nil {
		return nil, fmt.Errorf("error while getting link: %w", err)
	}

	// Parse the CIDR to get the network range.
	_, cidrNet, err := net.ParseCIDR(opts.CIDR)
	if err != nil {
		return nil, fmt.Errorf("error parsing CIDR: %w", err)
	}

	// Calculate the first usable IP in the range (network + 1).
	serverIP := make(net.IP, len(cidrNet.IP))
	copy(serverIP, cidrNet.IP)
	serverIP[len(serverIP)-1] += 1 // Increment last octet to get .1

	// Create server address with /32 (single host).
	serverAddr := fmt.Sprintf("%s/32", serverIP.String())
	addr, err := netlink.ParseAddr(serverAddr)
	if err != nil {
		return nil, fmt.Errorf("error parsing server address: %w", err)
	}

	// Add the server address to the interface.
	if err = netlink.AddrAdd(link, addr); err != nil {
		return nil, fmt.Errorf("error assigning server address: %w", err)
	}

	// Start the interface.
	if err = netlink.LinkSetUp(link); err != nil {
		return nil, fmt.Errorf("error bringing up the link: %w", err)
	}

	// Decode the private key.
	pk, err := encodeBase64ToHex(opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding private key: %w", err)
	}

	// Create a new wg device.
	lvl := device.LogLevelSilent
	if opts.Verbose {
		lvl = device.LogLevelVerbose
	}
	dev := device.NewDevice(t, conn.NewDefaultBind(), device.NewLogger(lvl, "(arbok) "))

	// Set the server config.
	serverConf := bytes.NewBuffer(nil)
	fmt.Fprintf(serverConf, "private_key=%s\n", pk)
	fmt.Fprintf(serverConf, "listen_port=%d\n", opts.ListenPort)
	if err = dev.IpcSetOperation(bufio.NewReader(serverConf)); err != nil {
		return nil, fmt.Errorf("error sending config to wg device: %w", err)
	}

	// Calculate public key from private key
	pubKey, err := privateKeyToPublicKey(opts.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error calculating public key: %w", err)
	}

	return &Tunnel{
		logger:     opts.Logger,
		dev:        dev,
		privateKey: opts.PrivateKey,
		publicKey:  pubKey,
	}, nil
}

// Up starts the wireguard device. This is a blocking call
// and waits till the context is cancelled.
func (tun *Tunnel) Up(ctx context.Context) error {
	// Start the device.
	if err := tun.dev.Up(); err != nil {
		return fmt.Errorf("error starting wg device: %w", err)
	}
	// Whenever context is cancelled, quit.
	<-ctx.Done()
	return nil
}

// GetPublicKey returns the server's public key
func (tun *Tunnel) GetPublicKey() string {
	return tun.publicKey
}

// AddPeer adds a new peer to the WireGuard interface
func (tun *Tunnel) AddPeer(publicKey, allowedIP string) error {
	// Decode the public key
	pk, err := encodeBase64ToHex(publicKey)
	if err != nil {
		return fmt.Errorf("error decoding public key: %w", err)
	}

	// Create peer config
	peerConf := bytes.NewBuffer(nil)
	fmt.Fprintf(peerConf, "public_key=%s\n", pk)
	fmt.Fprintf(peerConf, "allowed_ip=%s/32\n", allowedIP)
	
	if err := tun.dev.IpcSetOperation(bufio.NewReader(peerConf)); err != nil {
		return fmt.Errorf("error adding peer: %w", err)
	}

	tun.logger.Info("added peer", "public_key", publicKey[:8]+"...", "allowed_ip", allowedIP)
	return nil
}

// RemovePeer removes a peer from the WireGuard interface
func (tun *Tunnel) RemovePeer(publicKey string) error {
	// Decode the public key
	pk, err := encodeBase64ToHex(publicKey)
	if err != nil {
		return fmt.Errorf("error decoding public key: %w", err)
	}

	// Create peer removal config
	peerConf := bytes.NewBuffer(nil)
	fmt.Fprintf(peerConf, "public_key=%s\n", pk)
	fmt.Fprintf(peerConf, "remove=true\n")
	
	if err := tun.dev.IpcSetOperation(bufio.NewReader(peerConf)); err != nil {
		return fmt.Errorf("error removing peer: %w", err)
	}

	tun.logger.Info("removed peer", "public_key", publicKey[:8]+"...")
	return nil
}

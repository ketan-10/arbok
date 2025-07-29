package api

import (
	"fmt"
	"time"

	"github.com/mr-karan/arbok/internal/tunnel"
)

// generateTunnelScript generates a self-executing bash script for tunnel management
func (s *Server) generateTunnelScript(t *tunnel.Info) string {
	tunnelURL := fmt.Sprintf("https://%s.%s", t.Subdomain, s.cfg.Domain)
	
	// Generate the WireGuard config using the same method as manual provision
	wgConfig := s.generateWireGuardConfig(t)
	
	return fmt.Sprintf(`#!/bin/bash
set -e

# Arbok Tunnel Script
# Generated: %s
# Expires: %s (in %s)
# Tunnel: %s → localhost:%d

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TUNNEL_ID="%s"
TUNNEL_URL="%s"
LOCAL_PORT=%d
CONFIG_FILE="/tmp/arbok$$.conf"

# Cleanup function
cleanup() {
    echo -e "\n${RED}🔴 Stopping tunnel...${NC}"
    
    # Extract interface name from config file
    INTERFACE_NAME=$(basename "$CONFIG_FILE" .conf)
    
    # Stop WireGuard tunnel
    if sudo wg-quick down "$CONFIG_FILE" 2>/dev/null; then
        echo -e "${GREEN}✅ Tunnel stopped successfully${NC}"
    else
        echo -e "${YELLOW}⚠️  Attempting manual cleanup...${NC}"
        # Manual cleanup if wg-quick fails
        sudo ip link delete "$INTERFACE_NAME" 2>/dev/null || true
    fi
    
    # Remove config file
    rm -f "$CONFIG_FILE"
    echo -e "${GREEN}🧹 Cleanup complete${NC}"
    
    exit 0
}

# Setup signal handlers for graceful shutdown
trap cleanup INT TERM

# Check if running as root or with sudo
if [[ $EUID -ne 0 ]]; then
    echo -e "${RED}❌ This script requires root privileges${NC}"
    echo -e "Please run: ${BLUE}curl -s https://%s/start/%d | sudo bash${NC}"
    exit 1
fi

# Check if WireGuard is installed
if ! command -v wg-quick &> /dev/null; then
    echo -e "${RED}❌ WireGuard is not installed${NC}"
    echo -e "Please install WireGuard first:"
    echo -e "  Ubuntu/Debian: ${BLUE}apt install wireguard${NC}"
    echo -e "  CentOS/RHEL:   ${BLUE}yum install wireguard-tools${NC}"
    echo -e "  macOS:         ${BLUE}brew install wireguard-tools${NC}"
    exit 1
fi

echo -e "${BLUE}🐍 Starting Arbok tunnel...${NC}"

# Clean up any existing conflicting routes and interfaces
echo -e "${YELLOW}🧹 Cleaning up any existing tunnel routes and interfaces...${NC}"
sudo ip route del 10.100.0.1/32 2>/dev/null || true

# Clean up any existing arbok interfaces
for iface in $(sudo wg show interfaces 2>/dev/null | grep "^arbok" || true); do
    echo -e "${YELLOW}🔧 Removing existing interface: $iface${NC}"
    sudo wg-quick down "/tmp/$iface.conf" 2>/dev/null || sudo ip link delete "$iface" 2>/dev/null || true
done

# Create WireGuard configuration with proper permissions
cat > "$CONFIG_FILE" << 'EOF'
%s
EOF

# Set proper permissions (readable only by root)
chmod 600 "$CONFIG_FILE"

# Start the tunnel
echo -e "${YELLOW}⚡ Bringing up WireGuard interface...${NC}"
if sudo wg-quick up "$CONFIG_FILE"; then
    echo -e "${GREEN}✅ Tunnel active!${NC}"
    echo ""
    echo -e "${GREEN}🌐 Your tunnel URL:${NC} ${BLUE}%s${NC}"
    echo -e "${GREEN}📡 Forwarding:${NC} localhost:%d → %s"
    echo -e "${GREEN}⏱️  Expires:${NC} %s (in %s)"
    echo ""
    echo -e "${YELLOW}💡 Your local service should be running on localhost:%d${NC}"
    echo -e "${YELLOW}🛑 Press Ctrl+C to stop the tunnel${NC}"
    echo ""
else
    echo -e "${RED}❌ Failed to start tunnel${NC}"
    rm -f "$CONFIG_FILE"
    exit 1
fi

# Keep the script running and handle cleanup
echo -e "${GREEN}🟢 Tunnel is active${NC}"
echo -e "${YELLOW}💡 Keep this terminal open. Press Ctrl+C to stop the tunnel.${NC}"
echo ""

# Simple wait loop that can be interrupted
while true; do
    sleep 5
done
`, 
		t.CreatedAt.Format(time.RFC3339),
		t.ExpiresAt.Format(time.RFC3339), 
		t.TTL().Round(time.Minute),
		tunnelURL,
		t.Port,
		t.ID,
		tunnelURL,
		t.Port,
		s.cfg.Domain,
		t.Port,
		wgConfig,
		tunnelURL,
		t.Port,
		tunnelURL,
		t.ExpiresAt.Format("Jan 02, 15:04 MST"),
		t.TTL().Round(time.Minute),
		t.Port,
	)
}
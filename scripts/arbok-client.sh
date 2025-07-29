#!/bin/bash
# Arbok Client Helper Script
# A simple wrapper around wg-quick for better UX

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
ARBOK_SERVER="${ARBOK_SERVER:-http://localhost:8080}"
CONFIG_DIR="${HOME}/.arbok"
API_KEY="${ARBOK_API_KEY:-}"

# Ensure config directory exists
mkdir -p "$CONFIG_DIR"

# Helper functions
show_help() {
    echo "Arbok Client - Simple tunnel management"
    echo ""
    echo "Usage: $(basename $0) <command> [options]"
    echo ""
    echo "Commands:"
    echo "  start <port>    Start a tunnel for the specified port"
    echo "  stop [name]     Stop a tunnel (or all tunnels if no name given)"
    echo "  list            List active tunnels"
    echo "  help            Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  ARBOK_SERVER    Server URL (default: http://localhost:8080)"
    echo "  ARBOK_API_KEY   API key for authentication (optional)"
    echo ""
    echo "Examples:"
    echo "  $(basename $0) start 3000"
    echo "  $(basename $0) stop"
    echo "  $(basename $0) list"
}

start_tunnel() {
    local port=$1
    
    if [[ -z "$port" ]]; then
        echo -e "${RED}Error: Port number required${NC}"
        echo "Usage: $(basename $0) start <port>"
        exit 1
    fi
    
    # Validate port
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [ "$port" -lt 1 ] || [ "$port" -gt 65535 ]; then
        echo -e "${RED}Error: Invalid port number${NC}"
        exit 1
    fi
    
    echo -e "${BLUE}Starting tunnel for port ${port}...${NC}"
    
    # Download config
    local config_file="$CONFIG_DIR/arbok-${port}.conf"
    local curl_opts=(-s)
    
    if [[ -n "$API_KEY" ]]; then
        curl_opts+=(-H "X-API-Key: $API_KEY")
    fi
    
    if ! curl "${curl_opts[@]}" "${ARBOK_SERVER}/${port}" > "$config_file"; then
        echo -e "${RED}Error: Failed to create tunnel${NC}"
        rm -f "$config_file"
        exit 1
    fi
    
    # Extract tunnel info from config
    local subdomain=$(grep -oP '(?<=Arbok tunnel active! Local port \d+ → https://)[^.]+' "$config_file" 2>/dev/null || echo "tunnel")
    local url=$(grep -oP 'https://[^ ]+' "$config_file" | head -1)
    
    # Start tunnel
    if sudo wg-quick up "$config_file"; then
        echo -e "${GREEN}✓ Tunnel started successfully!${NC}"
        echo -e "${GREEN}URL: ${BLUE}${url}${NC}"
        echo -e "${YELLOW}To stop: $(basename $0) stop ${subdomain}${NC}"
    else
        echo -e "${RED}Error: Failed to start tunnel${NC}"
        rm -f "$config_file"
        exit 1
    fi
}

stop_tunnel() {
    local name=$1
    
    if [[ -z "$name" ]]; then
        # Stop all tunnels
        echo -e "${YELLOW}Stopping all tunnels...${NC}"
        for conf in "$CONFIG_DIR"/arbok-*.conf; do
            if [[ -f "$conf" ]]; then
                sudo wg-quick down "$conf" 2>/dev/null || true
                rm -f "$conf"
            fi
        done
        echo -e "${GREEN}✓ All tunnels stopped${NC}"
    else
        # Stop specific tunnel
        local config_file="$CONFIG_DIR/${name}.conf"
        
        # If name doesn't include arbok- prefix, try with it
        if [[ ! -f "$config_file" ]] && [[ ! "$name" =~ ^arbok- ]]; then
            config_file="$CONFIG_DIR/arbok-${name}.conf"
        fi
        
        if [[ -f "$config_file" ]]; then
            echo -e "${YELLOW}Stopping tunnel ${name}...${NC}"
            if sudo wg-quick down "$config_file"; then
                rm -f "$config_file"
                echo -e "${GREEN}✓ Tunnel stopped${NC}"
            else
                echo -e "${RED}Error: Failed to stop tunnel${NC}"
                exit 1
            fi
        else
            echo -e "${RED}Error: Tunnel '${name}' not found${NC}"
            exit 1
        fi
    fi
}

list_tunnels() {
    echo -e "${BLUE}Active tunnels:${NC}"
    local found=false
    
    for conf in "$CONFIG_DIR"/arbok-*.conf; do
        if [[ -f "$conf" ]]; then
            found=true
            local name=$(basename "$conf" .conf)
            local port=$(echo "$name" | grep -oP '\d+$')
            local url=$(grep -oP 'https://[^ ]+' "$conf" | head -1)
            
            # Check if interface is actually up
            if sudo wg show interfaces 2>/dev/null | grep -q "$name"; then
                echo -e "  ${GREEN}●${NC} ${name} → ${url}"
            else
                echo -e "  ${RED}○${NC} ${name} (config exists but not running)"
            fi
        fi
    done
    
    if [[ "$found" == "false" ]]; then
        echo -e "  ${YELLOW}No active tunnels${NC}"
    fi
}

# Main command handling
case "${1:-help}" in
    start)
        start_tunnel "$2"
        ;;
    stop)
        stop_tunnel "$2"
        ;;
    list|ls)
        list_tunnels
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        echo -e "${RED}Error: Unknown command '${1}'${NC}"
        echo ""
        show_help
        exit 1
        ;;
esac
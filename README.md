# Arbok

Secure HTTP tunnels to localhost using WireGuard. Share your local development server instantly without signup or complex setup.

## Quick Start

Choose your preferred approach:

### 1. One-liner (fastest)
```bash
# Start tunnel and pipe config directly to WireGuard
curl -s https://arbok.mrkaran.dev/3000?format=oneliner | sudo wg-quick up /dev/stdin

# Stop with: sudo wg-quick down /dev/stdin (if still running)
```

### 2. Helper client (recommended)  
```bash
# Download and use the helper client
curl -O https://arbok.mrkaran.dev/client && chmod +x client

./client start 3000    # Start tunnel for port 3000
./client list          # List active tunnels  
./client stop          # Stop all tunnels
```

### 3. Manual config (full control)
```bash
# Get tunnel config and manage manually
curl https://arbok.mrkaran.dev/3000 > tunnel.conf
sudo wg-quick up ./tunnel.conf

# Stop the tunnel
sudo wg-quick down ./tunnel.conf
```

### 4. Interactive script (deprecated)
```bash
# Self-executing script with cleanup handlers
curl -s https://arbok.mrkaran.dev/start/3000 | sudo bash
# Press Ctrl+C to stop
```

## Installation

1. **Build from source:**
```bash
git clone https://github.com/mr-karan/arbok
cd arbok
make build
```

2. **Configure** (copy `config.sample.toml` to `config.toml`):
```toml
[app]
domain = "arbok.yourdomain.com"

[auth]
# Optional: Add API keys for authentication
api_keys = ["secret-key-1"]

[tunnel]
default_ttl = "24h"
cleanup_interval = "5m"

[server]
cidr = "10.100.0.0/24"
listen_port = 54321
private_key = "your-wireguard-private-key"

[http]
listen_addr = ":8080"
```

3. **Run** (requires root for WireGuard):
```bash
sudo ./bin/server.bin --config config.toml
```

## Testing

Test without DNS setup using Host headers:

```bash
# Start local service
python3 -m http.server 3000 &

# Create tunnel
curl http://localhost:8080/3000 > tunnel.conf
sudo wg-quick up ./tunnel.conf

# Test with Host header (replace subdomain from tunnel.conf)
curl -H "Host: your-subdomain.localhost" http://localhost:8080
```

## Features

**Simple & Secure**
- One command setup with automatic lifecycle management
- WireGuard encryption with modern cryptography
- No account needed - anonymous tunnels by default
- Self-hosted - complete control over your infrastructure

**Production Ready**
- Prometheus metrics at `/metrics`
- Automatic tunnel cleanup with configurable TTLs  
- Resource management prevents IP exhaustion
- WebSocket and SSE support

## API Usage

### Quick provisioning endpoints
```bash
# One-liner format (for piping to wg-quick)
curl -s https://arbok.mrkaran.dev/3000?format=oneliner

# Standard config with instructions  
curl https://arbok.mrkaran.dev/3000

# Helper client script
curl -O https://arbok.mrkaran.dev/client

# Interactive script (deprecated)
curl -s https://arbok.mrkaran.dev/start/3000 | sudo bash
```

### RESTful API (requires API key)
```bash
# Create tunnel
curl -X POST -H "X-API-Key: your-key" https://tunnel.yourdomain.com/api/tunnel/3000

# List tunnels
curl -H "X-API-Key: your-key" https://tunnel.yourdomain.com/api/tunnels

# Delete tunnel
curl -X DELETE -H "X-API-Key: your-key" https://tunnel.yourdomain.com/api/tunnel/{id}
```

## How It Works

```
Browser → HTTPS → Arbok Server → WireGuard Tunnel → Local Service
```

1. HTTP API allocates IP and generates Curve25519 keypair
2. WireGuard tunnel created with encrypted connection
3. Browser requests proxied through tunnel to localhost

For detailed technical internals, see [docs/internals.md](docs/internals.md).

### Client Requirements

- Linux/macOS with WireGuard installed
- `wg-quick` command available  
- sudo/root access for WireGuard interface

## Comparison

| Feature | Arbok | ngrok | Cloudflare Tunnel |
|---------|-------|-------|-------------------|
| No signup | ✓ | ✗ | ✗ |
| Self-hosted | ✓ | ✗ | ✗ |
| Modern crypto | ✓ | ✗ | ✓ |
| Zero client install | ✓ | ✗ | ✗ |

## Security

- End-to-end encryption via WireGuard
- Automatic tunnel expiration prevents orphaned connections
- IP address isolation between tunnels
- No persistent logs of tunneled traffic

## License

MIT License - see [LICENSE](LICENSE) for details.
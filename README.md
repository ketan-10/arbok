# Arbok

Secure HTTP tunnels to localhost using WireGuard. Share your local development server instantly without signup or complex setup.

## Quick Start

**Single command (recommended):**
```bash
# Replace 3000 with your local port
curl -s https://arbok.mrkaran.dev/start/3000 | sudo bash

# Your app is now live at https://random-name-1234.arbok.mrkaran.dev
# Press Ctrl+C to stop the tunnel
```

**Manual setup (advanced):**
```bash
# Get tunnel config and start manually
curl https://arbok.mrkaran.dev/3000 > tunnel.conf
sudo wg-quick up ./tunnel.conf

# Stop the tunnel
sudo wg-quick down ./tunnel.conf
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

### Enhanced provisioning (single command)
```bash
curl -s https://arbok.mrkaran.dev/start/3000 | sudo bash
```

### Simple provisioning (manual config)
```bash
curl https://arbok.mrkaran.dev/3000 > tunnel.conf
```

### RESTful API
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
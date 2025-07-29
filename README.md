# Arbok

Secure HTTP tunnels to localhost using WireGuard. Share your local development server instantly without signup or complex setup.

**🌐 Try it now: [arbok.mrkaran.dev](https://arbok.mrkaran.dev)**

## Quick Start

```bash
# 1. Get tunnel config (replace 3000 with your local port)
curl https://arbok.mrkaran.dev/3000 > wg.conf

# 2. Start tunnel
sudo wg-quick up ./wg.conf

# 3. Stop tunnel when done
sudo wg-quick down ./wg.conf
```

Your local service is now accessible at the HTTPS URL shown in the config file.

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
curl http://localhost:8080/3000 > wg.conf
sudo wg-quick up ./wg.conf

# Test with Host header (replace subdomain from wg.conf)
curl -H "Host: your-subdomain.localhost" http://localhost:8080
```

## Features

**Simple & Secure**
- Straightforward tunnel setup and management
- WireGuard encryption with modern cryptography
- No account needed - anonymous tunnels by default
- Self-hosted - complete control over your infrastructure

**Production Ready**
- Prometheus metrics at `/metrics`
- Automatic tunnel cleanup with configurable TTLs  
- Resource management prevents IP exhaustion
- WebSocket and SSE support

## API Usage

### Tunnel provisioning
```bash
# Get WireGuard config with instructions
curl https://arbok.mrkaran.dev/3000
```

### RESTful API (requires API key)
```bash
# Create tunnel
curl -X POST -H "X-API-Key: your-key" https://arbok.mrkaran.dev/api/tunnel/3000

# List tunnels
curl -H "X-API-Key: your-key" https://arbok.mrkaran.dev/api/tunnels

# Delete tunnel
curl -X DELETE -H "X-API-Key: your-key" https://arbok.mrkaran.dev/api/tunnel/{id}
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
| Open source | ✓ | ✗ | ✗ |

## Security

- End-to-end encryption via WireGuard
- Automatic tunnel expiration prevents orphaned connections
- IP address isolation between tunnels
- No persistent logs of tunneled traffic

## License

MIT License - see [LICENSE](LICENSE) for details.
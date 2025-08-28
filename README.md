# Arbok

### Note following flow shows how "Kernal-mode" tcp stack and "User-mode" wireguard works. 
(but in arbok, fully tcp stack & wireguard are "user-mode")
```md
INCOMING (someone connecting to 10.100.0.2:3000):
1. **Physical NIC:**        UDP packet arrives (encrypted WG)
2. **Kernel -> Userspace:** wireguard-go reads UDP socket
3. **Userspace:**           wireguard-go decrypts → gets IP packet
4. **Userspace -> Kernel:** Writes decrypted packet to TUN (burrow)
5. **Kernel:**              "Oh, 10.100.0.2 is local!" 
6. **Kernel:**              TCP stack processes (port 3000 check)
7. **Kernel -> Userspace:** To app (if listening) or RST
```

```md
RESPONSE PATH:
1. **Kernel:**           TCP RST packet created for (internal application)→10.100.0.1
2. **Kernel→Userspace:** wireguard-go reads from TUN
3. **Userspace:**        wireguard-go encrypts packet
4. **Userspace→Kernel:** Sends as UDP to peer
5. **Physical NIC:**     UDP packet goes out
```

- Incoming for Another Peer (10.100.0.1 → 10.100.0.3:3000)
```md
INCOMING (someone connecting to 10.100.0.3:3000):
1. **Physical NIC:**     UDP packet arrives (encrypted WG from 10.100.0.1)
2. **Kernel→Userspace:** wireguard-go reads UDP socket
3. **Userspace:**        wireguard-go decrypts → gets IP packet for 10.100.0.3
4. **Userspace→Kernel:** Writes decrypted packet to TUN (burrow)
5. **Kernel:**           Checks routing table "10.100.0.3 via dev burrow"
6. **Kernel→Userspace:** Packet goes BACK to wireguard-go through TUN!
7. **Userspace:**        wireguard-go checks AllowedIPs for 10.100.0.3
8. **Userspace:**        wireguard-go encrypts with 10.100.0.3's peer key
9. **Userspace→Kernel:** Sends as UDP to 10.100.0.3's endpoint
10. **Physical NIC:**    UDP packet goes out to different peer
```

Secure HTTP tunnels to localhost using WireGuard. Share your local development server instantly without signup or complex setup.

**🌐 Try it now: [arbok.mrkaran.dev](https://arbok.mrkaran.dev)**

## Quick Start

```bash
# 1. Get tunnel config (replace 3000 with your local port)
curl https://arbok.mrkaran.dev/3000 > burrow.conf

# 2. Start tunnel
sudo wg-quick up ./burrow.conf

# 3. Stop tunnel when done
sudo wg-quick down ./burrow.conf
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

3. **Run**:
```bash
./bin/server.bin --config config.toml
```

## Testing

Test without DNS setup using Host headers:

```bash
# Start local service
python3 -m http.server 3000 &

# Create tunnel
curl http://localhost:8080/3000 > burrow.conf
sudo wg-quick up ./burrow.conf

# Test with Host header (replace subdomain from burrow.conf)
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

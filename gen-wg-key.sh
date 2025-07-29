#!/bin/bash
# Generate and validate WireGuard private key for Arbok server

echo "Generating WireGuard private key..."
PRIVATE_KEY=$(wg genkey)
PUBLIC_KEY=$(echo "$PRIVATE_KEY" | wg pubkey)

# Validate the key is proper base64
if echo "$PRIVATE_KEY" | base64 -d >/dev/null 2>&1; then
    echo "✓ Key validation passed"
else
    echo "✗ Key validation failed"
    exit 1
fi

# Check key length (should be 32 bytes when decoded)
KEY_LENGTH=$(echo "$PRIVATE_KEY" | base64 -d | wc -c)
if [ "$KEY_LENGTH" -eq 32 ]; then
    echo "✓ Key length correct (32 bytes)"
else
    echo "✗ Key length incorrect (got $KEY_LENGTH bytes, expected 32)"
    exit 1
fi

echo ""
echo "=== WireGuard Keys Generated ==="
echo "Private Key: $PRIVATE_KEY"
echo "Public Key:  $PUBLIC_KEY"
echo ""
echo "Add this exact private key to your docker-compose.yml:"
echo "private_key = \"$PRIVATE_KEY\""
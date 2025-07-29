package registry

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	
	"golang.org/x/crypto/curve25519"
)

// KeyGenerator generates cryptographic keys
type KeyGenerator interface {
	Generate() (privateKey, publicKey string, err error)
}

// NameGenerator generates human-friendly names
type NameGenerator interface {
	Generate() string
}

// WireGuardKeyGenerator generates WireGuard-compatible keys
type WireGuardKeyGenerator struct{}

func (g *WireGuardKeyGenerator) Generate() (privateKey, publicKey string, err error) {
	// Generate private key
	var priv [32]byte
	if _, err := rand.Read(priv[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Clamp private key (WireGuard requirement)
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	
	// Generate public key
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)
	
	privateKey = base64.StdEncoding.EncodeToString(priv[:])
	publicKey = base64.StdEncoding.EncodeToString(pub[:])
	
	return privateKey, publicKey, nil
}

// FriendlyNameGenerator generates memorable subdomain names
type FriendlyNameGenerator struct{}

func (g *FriendlyNameGenerator) Generate() string {
	adjectives := []string{
		"happy", "sunny", "bright", "swift", "calm", 
		"cool", "warm", "quick", "smart", "fresh",
		"clear", "light", "smooth", "sharp", "clean",
	}
	
	nouns := []string{
		"cloud", "wave", "star", "moon", "wind", 
		"rain", "snow", "fire", "lake", "tree",
		"river", "mountain", "valley", "ocean", "forest",
	}
	
	// Generate random indices
	var buf [3]byte
	rand.Read(buf[:])
	
	adj := adjectives[int(buf[0])%len(adjectives)]
	noun := nouns[int(buf[1])%len(nouns)]
	num := (int(buf[2])<<8 | int(buf[0])) % 10000
	
	return fmt.Sprintf("%s-%s-%04d", adj, noun, num)
}
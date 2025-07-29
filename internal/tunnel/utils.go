package tunnel

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	
	"golang.org/x/crypto/curve25519"
)

func encodeBase64ToHex(key string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	if len(decoded) != 32 {
		return "", errors.New("invalid key")
	}
	return hex.EncodeToString(decoded), nil
}

func privateKeyToPublicKey(privateKeyBase64 string) (string, error) {
	// Decode private key
	privBytes, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return "", err
	}
	if len(privBytes) != 32 {
		return "", errors.New("invalid private key length")
	}

	// Generate public key
	var priv, pub [32]byte
	copy(priv[:], privBytes)
	curve25519.ScalarBaseMult(&pub, &priv)

	return base64.StdEncoding.EncodeToString(pub[:]), nil
}

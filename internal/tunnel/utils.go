package tunnel

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
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

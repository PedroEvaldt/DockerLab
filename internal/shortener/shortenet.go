package shortener

import (
	"crypto/rand"
	"encoding/base64"
)

func Generate(n int) (string, error) {
	nBytes := base64.RawURLEncoding.DecodedLen(n) + 1
	buf := make([]byte, nBytes)
	_, err := rand.Read(buf)
	if err != nil {
		return "", err
	}
	s := base64.RawURLEncoding.EncodeToString(buf)
	return s[:n], nil
}

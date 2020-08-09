package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gtank/cryptopasta"
	"io"
	"strings"
)

// NewRandomKey generates a random 32 byte key.
func NewRandomKey() (string, error) {
	key := &[33]byte{} // slightly longer than we need to be safe
	_, err := io.ReadFull(rand.Reader, key[:])
	return base64.RawURLEncoding.EncodeToString(key[:]), err
}

// Decrypt is the inverse of encrypt, checking the HMAC and decrpyting the
// encoded data, if possible.
func Decrypt(encoded, key, sig string) ([]byte, error) {
	// convert our keys into bytes
	rawkey, err := toKey(key)
	if err != nil {
		return nil, err
	}

	rawsig, err := toKey(sig)
	if err != nil {
		return nil, err
	}

	// split into cyphertext & signature
	bits := strings.SplitN(encoded, ".", 2)
	if len(bits) != 2 {
		return nil, fmt.Errorf("decryption failed, encoded string invalid")
	}

	cypher, err := base64.RawURLEncoding.DecodeString(bits[0])
	if err != nil {
		return nil, err
	}

	signature, err := base64.RawURLEncoding.DecodeString(bits[1])
	if err != nil {
		return nil, err
	}

	if !cryptopasta.CheckHMAC(cypher, signature, rawsig) {
		return nil, fmt.Errorf("signature validation failed")
	}

	return cryptopasta.Decrypt(cypher, rawkey)
}

// Encrypt encrypts & base64 encodes the result into a string.
// It also attaches a HMAC signature on the end.
func Encrypt(plaintext []byte, key, sig string) (string, error) {
	// convert our keys into bytes
	rawkey, err := toKey(key)
	if err != nil {
		return "", err
	}

	rawsig, err := toKey(sig)
	if err != nil {
		return "", err
	}

	// encrypt & generate signature
	cyphertext, err := cryptopasta.Encrypt(plaintext, rawkey)
	if err != nil {
		return "", err
	}

	signature := cryptopasta.GenerateHMAC(cyphertext, rawsig)

	// smoosh together and we're done
	return fmt.Sprintf(
		"%s.%s",
		base64.RawURLEncoding.EncodeToString(cyphertext),
		base64.RawURLEncoding.EncodeToString(signature),
	), nil
}

// toKey transforms a string of at least len 32 into *[32]byte, as needed by
// cryptopasta library.
func toKey(s string) (*[32]byte, error) {
	if len(s) < 32 {
		return nil, fmt.Errorf("key too short for encryption/signing operation, want at least 32 chars.")
	}
	data := &[32]byte{}
	copy([]byte(s), data[:])
	return data, nil
}

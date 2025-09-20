package utility

import (
	"encoding/hex"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEncryptUnknownAlgorithm(t *testing.T) {
	_, err := Encrypt("unknown", "00112233", []byte("data"))
	assert.ErrorContains(t, err, "unknown encryption algorithm")
}

func TestDecryptUnknownAlgorithm(t *testing.T) {
	_, err := Decrypt("unknown", "00112233", []byte("data"))
	assert.ErrorContains(t, err, "unknown encryption algorithm")
}

func TestEncryptDecryptAESGCM(t *testing.T) {
	// Generate a 32-byte key in hex (AES-256)
	keyBytes := make([]byte, 32)
	for i := range keyBytes {
		keyBytes[i] = byte(i + 1)
	}
	hexKey := hex.EncodeToString(keyBytes)

	plaintext := []byte("this is a secret message")

	ciphertext, err := Encrypt(EncryptionAlgorithmAESGCM, hexKey, plaintext)
	assert.NilError(t, err)
	assert.Assert(t, len(ciphertext) > 0)

	decrypted, err := Decrypt(EncryptionAlgorithmAESGCM, hexKey, ciphertext)
	assert.NilError(t, err)
	assert.DeepEqual(t, decrypted, plaintext)
}

func TestDecryptAESGCMWithWrongKey(t *testing.T) {
	keyBytes := make([]byte, 32)
	for i := range keyBytes {
		keyBytes[i] = byte(i + 1)
	}
	hexKey := hex.EncodeToString(keyBytes)

	plaintext := []byte("secret")

	ciphertext, err := Encrypt(EncryptionAlgorithmAESGCM, hexKey, plaintext)
	assert.NilError(t, err)

	// Use a wrong key
	wrongKey := hex.EncodeToString([]byte("wrongwrongwrongwrongwrongwrong12"))
	_, err = Decrypt(EncryptionAlgorithmAESGCM, wrongKey, ciphertext)
	assert.ErrorContains(t, err, "failed to open and decrypt ciphertext")
}

package utility

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
)

// EncryptionAlgorithm represents the type of encryption algorithm to use.
type EncryptionAlgorithm string

const (
	// EncryptionAlgorithmAESGCM represents AES encryption in GCM mode.
	EncryptionAlgorithmAESGCM EncryptionAlgorithm = "aes-gcm"
)

// Encrypt encrypts the given plaintext using the specified encryption algorithm and key.
//
// Parameters:
//   - alg: the encryption algorithm to use (currently only "aes-gcm").
//   - key: the encryption key in hexadecimal string format.
//   - plaintext: the data to encrypt.
//
// Returns:
//   - []byte: the encrypted ciphertext.
//   - error: any error encountered during encryption.
//
// Example usage:
//
//	ciphertext, err := Encrypt(EncryptionAlgorithmAESGCM, hexKey, []byte("my secret"))
func Encrypt(alg EncryptionAlgorithm, key string, plaintext []byte) ([]byte, error) {
	switch alg {
	case EncryptionAlgorithmAESGCM:
		return aesGcmEncrypt(key, plaintext)
	default:
		return nil, errors.New("unknown encryption algorithm")
	}
}

// Decrypt decrypts the given ciphertext using the specified encryption algorithm and key.
//
// Parameters:
//   - alg: the encryption algorithm to use (currently only "aes-gcm").
//   - key: the encryption key in hexadecimal string format.
//   - ciphertext: the data to decrypt.
//
// Returns:
//   - []byte: the decrypted plaintext.
//   - error: any error encountered during decryption.
//
// Example usage:
//
//	plaintext, err := Decrypt(EncryptionAlgorithmAESGCM, hexKey, ciphertext)
func Decrypt(alg EncryptionAlgorithm, key string, ciphertext []byte) ([]byte, error) {
	switch alg {
	case EncryptionAlgorithmAESGCM:
		return aesGcmDecrypt(key, ciphertext)
	default:
		return nil, errors.New("unknown encryption algorithm")
	}
}

func aesGcmEncrypt(key string, plaintext []byte) ([]byte, error) {
	bytesKey, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %v", err)
	}

	block, err := aes.NewCipher(bytesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create block cipher: %v", err)
	}

	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm cipher: %v", err)
	}

	return aead.Seal(nil, nil, plaintext, nil), nil
}

func aesGcmDecrypt(key string, ciphertext []byte) ([]byte, error) {
	bytesKey, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("failed to decode key: %v", err)
	}

	block, err := aes.NewCipher(bytesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create block cipher: %v", err)
	}

	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm cipher: %v", err)
	}

	decrypted, err := aead.Open(nil, nil, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open and decrypt ciphertext: %v", err)
	}

	return decrypted, nil
}

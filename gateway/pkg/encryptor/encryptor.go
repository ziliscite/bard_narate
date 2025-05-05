package encryptor

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

var (
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

type Encryptor struct {
	encKey  []byte
	hmacKey []byte
}

func NewEncryptor(key string) (*Encryptor, error) {
	masterKey := []byte(key)
	if len(masterKey) != 64 {
		return nil, errors.New("invalid key length")
	}

	encKey := make([]byte, 32)
	hmacKey := make([]byte, 32)

	copy(encKey, masterKey[:32])
	copy(hmacKey, masterKey[32:64])

	return &Encryptor{encKey: encKey, hmacKey: hmacKey}, nil
}

func (en Encryptor) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(en.encKey)
	if err != nil {
		return "", fmt.Errorf("cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("GCM creation failed: %w", err)
	}

	mac := hmac.New(sha256.New, en.hmacKey)
	mac.Write([]byte(plaintext))
	nonce := mac.Sum(nil)[:gcm.NonceSize()]

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (en Encryptor) Decrypt(encrypted string) ([]byte, error) {
	if encrypted == "" {
		return nil, fmt.Errorf("%w: cannot decrypt empty string", ErrInvalidCiphertext)
	}

	ciphertext, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("%w: base64 decode failed: %w", ErrInvalidCiphertext, err)
	}

	block, err := aes.NewCipher(en.encKey)
	if err != nil {
		return nil, fmt.Errorf("cipher creation failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("GCM creation failed: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	plaintext, err := gcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		switch {
		case err.Error() == "cipher: message authentication failed":
			return nil, ErrInvalidCiphertext
		default:
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
	}

	return plaintext, nil
}

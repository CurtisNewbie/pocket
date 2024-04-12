package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	mr "math/rand"
)

var (
	digits           = []rune("0123456789")
	_password []byte = nil
)

func InitPassword(tmppw string) {
	if len(tmppw) > 32 {
		panic("password can only have 32 byte")
	}
	_password = make([]byte, 32)
	for i := 0; i < 32; i++ {
		if i < len(tmppw) {
			_password[i] = tmppw[i]
		} else {
			break
		}
	}
}

func Encrypt0(s string) string {
	v, _ := Encrypt(s)
	return v
}

func Encrypt(s string) (string, error) {
	aesCipher, err := aes.NewCipher(_password)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher, %v", err)
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM, %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce, %v", err)
	}

	encrypted := gcm.Seal(nonce, nonce, []byte(s), nil)
	return hex.EncodeToString(encrypted), nil
}

func Decrypt0(s string) string {
	v, _ := Decrypt(s)
	return v
}

func Decrypt(s string) (string, error) {
	dec, _ := hex.DecodeString(s)

	aesCipher, err := aes.NewCipher(_password)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher, %v", err)
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM, %v", err)
	}

	nonce := dec[:gcm.NonceSize()]

	decrypted, err := gcm.Open(nil, nonce, dec[gcm.NonceSize():], nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt, %v", err)
	}
	return string(decrypted), nil
}

// generate randon str based on given length and given charset
func doRand(n int, set []rune) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = set[mr.Intn(len(set))]
	}
	return string(b)
}

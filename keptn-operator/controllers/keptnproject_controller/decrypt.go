package keptnproject_controller

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

func decryptSecret(secret string) (string, error) {
	data := strings.Split(secret, ":")

	if data[0] == "rsa" {
		pemPrivate, ok := os.LookupEnv("RSA_PRIVATE_KEY")
		if !ok {
			return "", fmt.Errorf("environment variable RSA_PRIVATE_KEY is not set, will not be able to decrypt secrets")
		}

		secret, err := decryptPrivatePEM(data[1], pemPrivate)
		if err != nil {
			return "", err
		}
		return secret, nil
	}
	return secret, nil
}

func decryptPrivatePEM(message string, key string) (string, error) {
	block, err := decodePem(key)
	if err != nil {
		return "", err
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}
	ct, err := rsaOaepDecrypt(message, *privateKey)
	if err != nil {
		return "", err
	}

	return ct, nil
}

func rsaOaepDecrypt(cipherText string, privKey rsa.PrivateKey) (string, error) {
	ct, _ := base64.StdEncoding.DecodeString(cipherText)
	label := []byte("OAEP Encrypted")
	rng := rand.Reader

	plaintext, err := rsa.DecryptOAEP(sha256.New(), rng, &privKey, ct, label)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func decodePem(key string) (*pem.Block, error) {
	keyraw, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(keyraw)

	return block, nil
}

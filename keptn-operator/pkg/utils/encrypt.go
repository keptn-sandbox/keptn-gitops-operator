package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"
)

func RSA_OAEP_Encrypt(secretMessage string, key rsa.PublicKey) string {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &key, []byte(secretMessage), label)
	if err != nil {
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func EncryptPublicPEM(message string) (string, error) {
	pemPrivate, ok := os.LookupEnv("RSA_PRIVATE_KEY")
	if !ok {
		return "", fmt.Errorf("environment variable RSA_PRIVATE_KEY is not set, will not be able to decrypt secrets")
	}
	block, err := decodePem(pemPrivate)

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", err
	}

	ct := RSA_OAEP_Encrypt(message, privateKey.PublicKey)

	return "rsa:" + ct, err
}

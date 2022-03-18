package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io/ioutil"
)

func RSA_OAEP_Encrypt(secretMessage string, key rsa.PublicKey) (string, error) {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &key, []byte(secretMessage), label)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func EncryptPublicPEM(message string, keyfile string) (string, error) {
	key, err := ioutil.ReadFile(keyfile) // just pass the file name
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(key)
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return "", err
	}

	ct, err := RSA_OAEP_Encrypt(message, *publicKey)
	if err != nil {
		return "", err
	}

	return ct, nil
}


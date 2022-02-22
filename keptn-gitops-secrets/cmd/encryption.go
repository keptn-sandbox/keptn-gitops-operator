package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
)

func RSA_OAEP_Encrypt(secretMessage string, key rsa.PublicKey) string {
	label := []byte("OAEP Encrypted")
	rng := rand.Reader
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rng, &key, []byte(secretMessage), label)
	if err != nil {
		fmt.Println("Test")
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(ciphertext)
}

func EncryptPublicPEM(message string, keyfile string) string {
	key, err := ioutil.ReadFile(keyfile) // just pass the file name
	if err != nil {
		fmt.Print(err)
	}

	block,_ := pem.Decode(key)
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		fmt.Println(err)
	}

	ct := RSA_OAEP_Encrypt(message, *publicKey)

	return ct
}


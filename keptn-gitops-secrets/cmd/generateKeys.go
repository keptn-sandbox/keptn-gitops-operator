/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"

	"github.com/spf13/cobra"
)

type KeyGeneration interface {
	RunKeyGeneration() error
}

type keyGenerationImpl struct {
}

type KeyGenerationCmdParams struct {
	Basename *string
}

var keyGenerationParams *KeyGenerationCmdParams

func GenerateKeys(filebase string) (private, public string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	publicKey := &privateKey.PublicKey

	privkey_bytes := x509.MarshalPKCS1PrivateKey(privateKey)
	private = string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privkey_bytes,
		},
	))

	pubkey_bytes := x509.MarshalPKCS1PublicKey(publicKey)
	public = string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PUBLIC KEY",
			Bytes: pubkey_bytes,
		},
	))
	ioutil.WriteFile(filebase+".pub", []byte(public), 0644)
	ioutil.WriteFile(filebase+".priv", []byte(private), 0644)
	return private, public
}

func (keygeneration *keyGenerationImpl) RunKeyGeneration() error {
	private, public := GenerateKeys(*keyGenerationParams.Basename)

	fmt.Println(private)
	fmt.Println(public)

	return nil
}

func NewKeyGeneration(keyGeneration KeyGeneration) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-keys",
		Short: `Creates a new keypair`,
		Long:  `Creates a new key pair`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = keyGeneration.RunKeyGeneration()

			return nil
		},
	}

	keyGenerationParams = &KeyGenerationCmdParams{}
	keyGenerationParams.Basename = cmd.Flags().StringP("filename", "f", "", "Name of the key-pair")
	return cmd
}

func init() {
	keyGeneration := &keyGenerationImpl{}
	keyGenerationCmd := NewKeyGeneration(keyGeneration)
	rootCmd.AddCommand(keyGenerationCmd)
}

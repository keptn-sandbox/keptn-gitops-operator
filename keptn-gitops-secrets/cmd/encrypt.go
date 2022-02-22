package cmd

import (
	"fmt"
	//TODO: lib approval
	"github.com/spf13/cobra"
)


//go:generate mockgen -source=triggerDeploy.go -destination=deployment_mock.go -package=cmd Deployment

type Encryption interface {
	RunEncryption() error
}

type encryptionImpl struct {
}

type EncryptionCmdParams struct {
	PublicKey *string
	Secret *string
}

var encryptionParams *EncryptionCmdParams

func (encryption *encryptionImpl) RunEncryption() error {
	fmt.Println(EncryptPublicPEM( *encryptionParams.Secret, *encryptionParams.PublicKey))
	return nil
}

func NewEncryptCmd(encryption Encryption) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "encrypt",
		Short: `Encrypts a secret using a given public key`,
		Long: `Encrypts a secret using a given public key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = encryption.RunEncryption()

			return nil
		},
	}

	encryptionParams = &EncryptionCmdParams{}
	encryptionParams.PublicKey = cmd.Flags().StringP("public-key", "p", "", "The path to the public key")
	encryptionParams.Secret = cmd.Flags().StringP("secret", "s", "", "The unencrypted secret")

	cmd.MarkFlagRequired("public-key")
	cmd.MarkFlagRequired("secret")

	return cmd
}

func init() {
	encryption := &encryptionImpl{}
	encryptCmd := NewEncryptCmd(encryption)
	rootCmd.AddCommand(encryptCmd)
}

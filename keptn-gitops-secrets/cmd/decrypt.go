package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

//go:generate mockgen -source=triggerDeploy.go -destination=deployment_mock.go -package=cmd Deployment

type Decryption interface {
	RunDecryption() error
}

type decryptionImpl struct {
}

type DecryptionCmdParams struct {
	PrivateKey *string
	Secret     *string
}

var decryptionParams *DecryptionCmdParams

func (decryption *decryptionImpl) RunDecryption() error {
	fmt.Println(decryptPrivatePEM(*decryptionParams.Secret, *decryptionParams.PrivateKey))
	return nil
}

func NewDecryptCmd(decryption Decryption) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt",
		Short: `Decrypts a secret using a given private key`,
		Long:  `Decrypts a secret using a given private key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = decryption.RunDecryption()

			return nil
		},
	}

	decryptionParams = &DecryptionCmdParams{}
	decryptionParams.PrivateKey = cmd.Flags().StringP("private-key", "p", "", "The path to the private key")
	decryptionParams.Secret = cmd.Flags().StringP("secret", "s", "", "The unencrypted secret")

	cmd.MarkFlagRequired("private-key")
	cmd.MarkFlagRequired("secret")

	return cmd
}

func init() {
	decryption := &decryptionImpl{}
	decryptCmd := NewDecryptCmd(decryption)
	rootCmd.AddCommand(decryptCmd)
}

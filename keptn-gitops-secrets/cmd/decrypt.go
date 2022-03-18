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
	secret, err := decryptPrivatePEM(*decryptionParams.Secret, *decryptionParams.PrivateKey)
	if err != nil {
		return err
	}
	if !quiet {
		fmt.Println(secret)
	}
	return nil
}

func NewDecryptCmd(decryption Decryption) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt",
		Short: `Decrypts a secret using a given private key`,
		Long:  `Decrypts a secret using a given private key`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := decryption.RunDecryption()
			res := Result{}
			if err != nil {
				if jsonEnabled {
					res.Message = err.Error()
					res.Result = false
					printJsonResult(res)
				}
				return err
			}
			if jsonEnabled {
				res.Message = "The String has been encrypted successfully"
				res.Result = true
				printJsonResult(res)
			}
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

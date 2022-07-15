package cmd

import (
	"embed"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"log"
	"os"
	"strings"
)

//go:generate mockgen -source=triggerDeploy.go -destination=deployment_mock.go -package=cmd Deployment

type Bootstrap interface {
	RunBootstrap() error
}

type bootstrapImpl struct {
}

type BootstrapCmdParams struct {
	PrivateKey *string
	Secret     *string
}

var bootstrapParams *BootstrapCmdParams

type promptContent struct {
	errorMsg     string
	label        string
	defaultValue string
}

type selectContent struct {
	label      string
	selections []string
}

func (bootstrap *bootstrapImpl) RunBootstrap() error {

	secret, err := decryptPrivatePEM(*decryptionParams.Secret, *decryptionParams.PrivateKey)
	if err != nil {
		return err
	}
	if !quiet {
		fmt.Println(secret)
	}
	return nil
}

type Input struct {
	Project              string
	ApiUrl               string
	ApiTokenType         string
	ApiToken             string
	ConfigRepoUrl        string
	ConfigRepoUsername   string
	ConfigRepoToken      string
	ConfigRepoBranch     string
	ConfigRepoPath       string
	UpstreamRepoUrl      string
	UpstreamRepoUsername string
	UpstreamRepoToken    string
	UpstreamRepoBranch   string
	EncUpstreamToken     string
	EncConfigRepoToken   string
	EncToken             string
	EncPrivateKey        string
	GenerateSetupScript  bool
	GitOpsVersion        string
	Stages               []string
	Services             []string
	InitShipyard         string
	DeploymentEvent      string
}

type StageTemplate struct {
	Stages  []string
	Project string
}

//go:embed tpl
var templateFiles embed.FS

var input Input

func NewBootstrapCmd(bootstrap Bootstrap) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: `Bootstraps a new GitOps Project`,
		Long:  `Bootstraps a new GitOps Project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runBootstrap()
			if err != nil {
				log.Fatal(err)
			}

			_ = bootstrap.RunBootstrap
			return nil
		},
	}

	return cmd
}

func init() {
	bootstrap := &bootstrapImpl{}
	bootstrapCmd := NewBootstrapCmd(bootstrap)
	rootCmd.AddCommand(bootstrapCmd)
}

func promptGetInput(pc promptContent) string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New(pc.errorMsg)
		}
		return nil
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     pc.label,
		Templates: templates,
		Validate:  validate,
		Default:   pc.defaultValue,
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Input: %s\n", result)

	return result
}

func promptPassword(pc promptContent) string {
	validate := func(input string) error {
		if len(input) < 6 {
			return errors.New("Password must have more than 6 characters")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    pc.label,
		Validate: validate,
		Mask:     '*',
		Default:  pc.defaultValue,
	}

	result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	return result
}

func selectTokenType(sc selectContent) string {
	prompt := promptui.Select{
		Label: sc.label,
		Items: sc.selections,
	}

	_, result, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	return result
}

func promptBoolean(sc selectContent) bool {
	prompt := promptui.Select{
		Label: sc.label,
		Items: []string{"Yes", "No"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("Prompt failed %v\n", err)
	}
	return result == "Yes"
}

func promptGetArray(pc promptContent) []string {
	validate := func(input string) error {
		if len(input) <= 0 {
			return errors.New(pc.errorMsg)
		}
		return nil
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     pc.label,
		Templates: templates,
		Validate:  validate,
		Default:   pc.defaultValue,
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		os.Exit(1)
	}

	cleanData := strings.ReplaceAll(result, " ", "")
	results := strings.Split(cleanData, ",")

	return results
}

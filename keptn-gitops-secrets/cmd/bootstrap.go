package cmd

import (
	"embed"
	"errors"
	"fmt"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"log"
	"os"
	"text/template"
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
			fmt.Println("======= General Keptn Configuration  =======")
			projectPromptContent := promptContent{"Please provide the Project Name", "Project Name", ""}
			input.Project = promptGetInput(projectPromptContent)

			instancePromptContent := promptContent{"Please provide the Instance URL", "Instance URL", "https://"}
			input.ApiUrl = promptGetInput(instancePromptContent)

			apiTokenPromptContent := promptContent{"Please provide the API Token", "API Token", ""}
			input.ApiToken = promptPassword(apiTokenPromptContent)

			apiTokenTypeSelectContent := selectContent{"Please provide the API Token type", []string{"x-token", "internal"}}
			input.ApiTokenType = selectTokenType(apiTokenTypeSelectContent)

			fmt.Println("======= Configuration Repository Configuration =======")
			keptnConfigRepoUsernamePromptContent := promptContent{"Please provide the Config Repo Username", "Config Repo Username", ""}
			input.ConfigRepoUsername = promptGetInput(keptnConfigRepoUsernamePromptContent)

			keptnConfigRepoPromptContent := promptContent{"Please provide the Config Repo URL", "Config Repo URL", "https://github.com/" + input.ConfigRepoUsername + "/" + input.Project}
			input.ConfigRepoUrl = promptGetInput(keptnConfigRepoPromptContent)

			keptnConfigRepoBranchPromptContent := promptContent{"Please provide the Config Repo Branch", "Config Repo Branch", "main"}
			input.ConfigRepoBranch = promptGetInput(keptnConfigRepoBranchPromptContent)

			keptnConfigRepoPathPromptContent := promptContent{"Please provide the Config Repo Path", "Config Repo Target Path", ".keptn"}
			input.ConfigRepoPath = promptGetInput(keptnConfigRepoPathPromptContent)

			keptnConfigRepoTokenPromptContent := promptContent{"Please provide the Config Repo Token", "Config Repo Token", ""}
			input.ConfigRepoToken = promptPassword(keptnConfigRepoTokenPromptContent)

			fmt.Println("======= Upstream Repository Configuration =======")
			keptnUpstreamRepoPromptContent := promptContent{"Please provide the Upstream Repo URL", "Upstream Repo URL", input.ConfigRepoUrl + "-upstream"}
			input.UpstreamRepoUrl = promptGetInput(keptnUpstreamRepoPromptContent)

			keptnUpstreamRepoUsernamePromptContent := promptContent{"Please provide the Upstream Repo Username", "Upstream Repo Username", input.ConfigRepoUsername}
			input.UpstreamRepoUsername = promptGetInput(keptnUpstreamRepoUsernamePromptContent)

			keptnUpstreamRepoTokenPromptContent := promptContent{"Please provide the Upstream Repo Token", "Upstream Repo Token", input.ConfigRepoToken}
			input.UpstreamRepoToken = promptPassword(keptnUpstreamRepoTokenPromptContent)

			keptnUpstreamRepoBranchPromptContent := promptContent{"Please provide the Upstream Repo Branch", "Upstream Repo Branch", "deploy"}
			input.UpstreamRepoBranch = promptGetInput(keptnUpstreamRepoBranchPromptContent)

			fmt.Println("======= Setting up tokens =======")
			if _, err := os.Stat("./" + input.Project + "-key.priv"); errors.Is(err, os.ErrNotExist) {
				fmt.Println("Will create keypair")
				_, _ = GenerateKeys(input.Project + "-key")
			} else {
				fmt.Println("Keypair already exists")
			}

			input.EncToken, _ = EncryptPublicPEM(input.ApiToken, input.Project+"-key.pub")
			input.EncUpstreamToken, _ = EncryptPublicPEM(input.UpstreamRepoToken, input.Project+"-key.pub")
			input.EncConfigRepoToken, _ = EncryptPublicPEM(input.ConfigRepoToken, input.Project+"-key.pub")

			if _, err := os.Stat("./out"); os.IsNotExist(err) {
				err := os.Mkdir("./out", 0700)
				if err != nil {
					log.Fatal(err)
				}
			}

			fmt.Println("\n======= Output: gitrepo.yaml =======")
			f, err := os.Create("./out/gitrepo.yaml")
			if err != nil {
				log.Println("create file: ", err)
			}
			defer f.Close()
			t := template.Must(template.ParseFS(templateFiles, "tpl/gitrepo.yaml"))
			err = t.Execute(f, input)
			if err != nil {
				log.Fatal(err)
			}

			fmt.Println("\n======= Output: project.yaml =======")
			g, err := os.Create("./out/project.yaml")
			if err != nil {
				log.Println("create file: ", err)
			}
			defer g.Close()
			t = template.Must(template.ParseFS(templateFiles, "tpl/project.yaml"))
			err = t.Execute(g, input)
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

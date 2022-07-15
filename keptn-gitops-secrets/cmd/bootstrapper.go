package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

func runBootstrap() error {
	outdir := os.Getenv("OUTPATH")
	if outdir == "" {
		outdir = "./out"
	}
	path, err := checkCreateDir(outdir)
	if err != nil {
		log.Fatal(err)
	}

	err = getData()
	if err != nil {
		log.Fatal(err)
	}

	projectPath, err := checkCreateDir(filepath.Join(path, input.Project))
	if err != nil {
		log.Fatal(err)
	}

	configPath, err := checkCreateDir(filepath.Join(projectPath, "config"))
	if err != nil {
		log.Fatal(err)
	}

	configBasePath, err := checkCreateDir(filepath.Join(configPath, "base"))
	if err != nil {
		log.Fatal(err)
	}

	err = renderTemplate(configPath, "base", "projectmeta.yaml", input.Project)
	if err != nil {
		log.Fatal(err)
	}

	configStageBasePath, err := checkCreateDir(filepath.Join(configPath, "stages"))
	if err != nil {
		log.Fatal(err)
	}

	for _, service := range input.Services {
		err = renderTemplate(configBasePath, service, "metadata.yaml", input.Project)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, stage := range input.Stages {
		configStagePath, err := checkCreateDir(filepath.Join(configStageBasePath, stage))
		if err != nil {
			log.Fatal(err)
		}

		for _, service := range input.Services {
			_, err := checkCreateDir(filepath.Join(configStagePath, service))
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	err = renderTemplate(projectPath, "setup", "gitrepo.yaml", input)
	if err != nil {
		log.Fatal(err)
	}

	err = renderTemplate(projectPath, "upstream", "shipyard.yaml", input)
	if err != nil {
		log.Fatal(err)
	}

	input.InitShipyard, err = encodeFile(projectPath, "upstream", "shipyard.yaml")
	if err != nil {
		log.Fatal(err)
	}

	for _, i := range []string{"project.yaml", "instance.yaml", "sequences.yaml"} {
		err = renderTemplate(projectPath, "config", i, input)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = renderTemplate(projectPath, "config", "stage.yaml", input)
	if err != nil {
		log.Fatal(err)
	}

	err = renderTemplate(projectPath, "config", "service.yaml", input)
	if err != nil {
		log.Fatal(err)
	}

	if input.GenerateSetupScript {
		err = renderTemplate(projectPath, "setup", "install.sh", input)
		if err != nil {
			log.Fatal(err)
		}
	}
	return nil
}

func getData() error {
	fmt.Println("======= General Keptn Configuration  =======")
	projectPromptContent := promptContent{"Please provide the Project Name", "Project Name", ""}
	input.Project = promptGetInput(projectPromptContent)

	instancePromptContent := promptContent{"Please provide the Instance URL", "Instance URL", "https://"}
	input.ApiUrl = promptGetInput(instancePromptContent)

	apiTokenTypeSelectContent := selectContent{"Please provide the API Token type", []string{"x-token", "internal"}}
	input.ApiTokenType = selectTokenType(apiTokenTypeSelectContent)

	apiTokenPromptContent := promptContent{"Please provide the API Token", "API Token", ""}
	input.ApiToken = promptPassword(apiTokenPromptContent)

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

	fmt.Println("======= Deployment Configuration =======")
	stagesPromptContent := promptContent{"Please provide your preferred stages", "Stages", "dev, hardening, production"}
	input.Stages = promptGetArray(stagesPromptContent)

	servicesPromptContent := promptContent{"Please provide your services", "Services", "demo-service"}
	input.Services = promptGetArray(servicesPromptContent)

	deployEventPromptContent := promptContent{"Which sequence do you want to use for deployments", "Deployment Sequence", "artifact-delivery"}
	input.DeploymentEvent = promptGetInput(deployEventPromptContent)

	optionCreateSetupScript := selectContent{"Do you want to generate a Setup Script?", nil}
	input.GenerateSetupScript = promptBoolean(optionCreateSetupScript)

	if input.GenerateSetupScript {
		gitOpsVersionPromptContent := promptContent{"Please provide the preferred GitOps Operator Version", "GitOps Operators Version", "0.1.0-pre.8"}
		input.GitOpsVersion = promptGetInput(gitOpsVersionPromptContent)
	}

	fmt.Println("======= Setting up tokens =======")
	var private string
	if _, err := os.Stat("./" + input.Project + "-key.priv"); errors.Is(err, os.ErrNotExist) {
		fmt.Println("Will create keypair")
		private, _ = GenerateKeys(input.Project + "-key")
	} else {
		file, err := ioutil.ReadFile(input.Project + "-key.priv") // just pass the file name
		if err != nil {
			return err
		}
		private = string(file)
		fmt.Println("Keypair already exists")
	}

	input.EncPrivateKey = base64.StdEncoding.EncodeToString([]byte(private))
	input.EncToken, _ = EncryptPublicPEM(input.ApiToken, input.Project+"-key.pub")
	input.EncUpstreamToken, _ = EncryptPublicPEM(input.UpstreamRepoToken, input.Project+"-key.pub")
	input.EncConfigRepoToken, _ = EncryptPublicPEM(input.ConfigRepoToken, input.Project+"-key.pub")

	return nil
}

func checkCreateDir(directory string) (string, error) {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err := os.Mkdir(directory, 0700)
		if err != nil {
			return "", err
		}
	}
	path, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	return path, nil
}

func renderTemplate(path string, subdir string, filename string, data interface{}) error {
	fullpath, err := checkCreateDir(filepath.Join(path, subdir))
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create(filepath.Join(fullpath, filename))
	if err != nil {
		return err
	}
	defer file.Close()

	t := template.Must(template.ParseFS(templateFiles, "tpl/"+filename))
	err = t.Execute(file, data)
	if err != nil {
		return err
	}
	return nil
}

func encodeFile(path string, subdir string, filename string) (string, error) {
	file, err := ioutil.ReadFile(filepath.Join(path, subdir, filename))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(file)), nil
}

package git

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	keptnutils "github.com/keptn/kubernetes-utils/pkg"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type GitCredentials struct {
	User      string `json:"user,omitempty"`
	Token     string `json:"token,omitempty"`
	RemoteURI string `json:"remoteURI,omitempty"`
}

//go:generate moq -pkg githandler_mock -skip-ensure -out ../eventhandler/fake/githandler_mock.go . GitHandlerInterface
type GitHandlerInterface interface {
	GetGitSecret(project string, namespace string) (GitCredentials, error)
	UpdateGitRepo(credentials GitCredentials, stage string, service string, version string) error
}

type GitHandler struct {
}

func (gh *GitHandler) GetGitSecret(project string, namespace string) (GitCredentials, error) {
	secret := GitCredentials{}
	clientset, err := keptnutils.GetClientset(true)
	if err != nil {
		return GitCredentials{}, err
	}

	gitSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "git-credentials-"+project, metav1.GetOptions{})
	if err != nil {
		return GitCredentials{}, err
	}

	err = json.Unmarshal(gitSecret.Data["git-credentials"], &secret)
	if err != nil {
		return GitCredentials{}, err
	}
	return secret, nil
}

func (gh *GitHandler) UpdateGitRepo(credentials GitCredentials, stage string, service string, version string) error {
	authentication := &http.BasicAuth{
		Username: credentials.User,
		Password: credentials.Token,
	}

	cloneOptionsMaster := git.CloneOptions{
		URL:           credentials.RemoteURI,
		Auth:          authentication,
		ReferenceName: plumbing.ReferenceName("refs/tags/" + service + "-" + version),
		SingleBranch:  true,
	}

	cloneOptionsStage := git.CloneOptions{
		URL:           credentials.RemoteURI,
		Auth:          authentication,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + stage),
		SingleBranch:  true,
	}

	commitOptions := git.CommitOptions{
		Author: &object.Signature{
			Name:  "Keptn Promotion Service",
			Email: "noreply@keptn.sh",
			When:  time.Now(),
		},
	}

	dirMaster, _ := ioutil.TempDir("", "temp_dir_master")
	dirStage, _ := ioutil.TempDir("", "temp_dir_"+stage)

	_, err := git.PlainClone(dirMaster, false, &cloneOptionsMaster)
	if err != nil {
		log.Println("Could not checkout "+credentials.RemoteURI+"/master", err)
		return err
	}

	stageRepo, err := git.PlainClone(dirStage, false, &cloneOptionsStage)
	if err != nil {
		log.Println("Could not checkout "+credentials.RemoteURI+"/"+stage, err)
		return err
	}

	w, err := stageRepo.Worktree()
	if err != nil {
		fmt.Printf("%v", err)
		return err
	}

	// Remove service directory
	os.RemoveAll(filepath.Join(dirStage, service))

	fs := afero.NewOsFs()

	err = mergeHelmValues(fs, service, stage, dirMaster, dirStage)
	if err != nil {
		log.Println("Couldn't Merge Helm Values", err)
	}

	err = performFileMove(fs, service, stage, dirMaster, dirStage)
	if err != nil {
		return err
	}

	// Replace Version in the Values File
	err = ReplaceInFile(filepath.Join(dirStage, service, "helm", service, "values.yaml"), "{{ keptn/ImageVersion }}", version)
	if err != nil {
		log.Println("Couldn't Replace Version in values.yaml")
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dirStage
	err = cmd.Run()
	if err != nil {
		log.Println("Could not add files")
	}

	_, err = w.Commit("Updated to version "+version, &commitOptions)
	if err != nil {
		log.Println("Couldn't commit "+stage, err)
	}

	err = stageRepo.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       authentication,
	})
	if err != nil {
		log.Println("Couldn't push "+stage, err)
	}
	defer os.RemoveAll(dirMaster)
	defer os.RemoveAll(dirStage)

	return nil
}

func mergeHelmValues(fs afero.Fs, serviceName, stageName, keptnGitSourceDir, keptnGitDestinationDir string) error {
	valuesSourceBase := filepath.Join(keptnGitSourceDir, "base", serviceName, "helm", serviceName, "values.yaml")
	valuesSourceStage := filepath.Join(keptnGitSourceDir, "stages", stageName, serviceName, "helm", serviceName, "values.yaml")

	fmt.Println("Merging values from " + valuesSourceBase + " with " + valuesSourceStage)
	err, values := MergeValues(valuesSourceBase, valuesSourceStage)
	if err != nil {
		return err
	}

	out, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("Could not create merged values file: %v", err)
	}
	err = afero.WriteFile(fs, filepath.Join(valuesSourceStage), out, 0644)
	if err != nil {
		return fmt.Errorf("Could not write values file in "+valuesSourceStage, err)
	}
	return nil
}

func performFileMove(fs afero.Fs, serviceName, stageName, keptnGitSourceDir, keptnGitDestinationDir string) error {

	serviceSourceDir := filepath.Join(keptnGitSourceDir, "base", serviceName)
	serviceSourceStageDir := filepath.Join(keptnGitSourceDir, "stages", stageName, serviceName)
	serviceDestinationDir := filepath.Join(keptnGitDestinationDir, serviceName)

	err := moveFiles(fs, serviceSourceDir, serviceDestinationDir, serviceName)
	if err != nil {
		return err
	}

	err = moveFiles(fs, serviceSourceStageDir, serviceDestinationDir, serviceName)
	if err != nil {
		return err
	}

	return nil
}

func moveFiles(fs afero.Fs, serviceSourceDir string, serviceDestinationDir string, serviceName string) error {

	// create empty source dir if necessary
	err := fs.MkdirAll(serviceSourceDir, os.ModePerm)
	if err != nil {
		return err
	}

	err = afero.Walk(fs, serviceSourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// find the first occurrence of service name:
		firstIndex := strings.Index(path, serviceName)
		if firstIndex < 0 {
			return errors.New("Service name " + serviceName + " not found in path " + path)
		}
		newSubFolderAndFileName := path[firstIndex+len(serviceName):]
		newFullFileName := filepath.Join(serviceDestinationDir, newSubFolderAndFileName)

		if info.IsDir() {

			log.Printf("Creating directory '%s'\n", newFullFileName)

			err := fs.MkdirAll(newFullFileName, 0700)
			if err != nil {
				return err
			}

		} else {

			log.Printf("Moving file %s from '%s' to '%s'\n", info.Name(), serviceSourceDir, serviceDestinationDir)

			// Move file from source to destination:
			err = fs.Rename(path, newFullFileName)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

func ReplaceInFile(file string, searchString string, replaceString string) error {
	input, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("Couldn't open file:, err")
	}

	output := bytes.Replace(input, []byte(searchString), []byte(replaceString), -1)

	if err = ioutil.WriteFile(file, output, 0666); err != nil {
		return fmt.Errorf("Couldn't write file:, err")
	}

	return nil
}

func MergeValues(orig string, stage string) (error, map[string]interface{}) {
	err, newValues := getValues(stage)
	if err != nil {
		log.Println("Had problems getting the stage values", err)
	}
	err, inputValues := getValues(orig)
	if err != nil {
		log.Println("Had problems getting the base values", err)
	}

	if inputValues != nil && newValues == nil {
		return nil, inputValues
	}

	if inputValues == nil && newValues != nil {
		return nil, newValues
	}

	if inputValues == nil && newValues == nil {
		return fmt.Errorf("No Values file specified"), nil
	}

	inputChart := chart.Chart{Values: inputValues}
	err = NewValuesManipulator(newValues).Manipulate(&inputChart)
	if err != nil {
		return err, nil
	}
	return nil, inputChart.Values
}

func getValues(file string) (error, map[string]interface{}) {
	inputFile, err := ioutil.ReadFile(file)
	if err != nil {
		return err, nil
	}
	values := map[string]interface{}{}
	if err := yaml.Unmarshal(inputFile, &values); err != nil {
		return err, nil
	}
	return nil, values
}

// ValuesManipulator allows to manipulate the values of a Helm chart
type ValuesManipulator struct {
	values map[string]interface{}
}

// NewValuesManipulator creates a new ValuesManipulator
func NewValuesManipulator(values map[string]interface{}) *ValuesManipulator {
	return &ValuesManipulator{
		values: values,
	}
}

// Manipulate updates the values
func (v *ValuesManipulator) Manipulate(ch *chart.Chart) error {

	// Change values
	for k, v := range v.values {
		// Merge ch.Values[k] in v
		merge(v, ch.Values[k])
		ch.Values[k] = v
	}
	return nil
}

func merge(in1, in2 interface{}) interface{} {
	switch in1 := in1.(type) {
	case []interface{}:
		in2, ok := in2.([]interface{})
		if !ok {
			return in1
		}
		return append(in1, in2...)
	case map[string]interface{}:
		in2, ok := in2.(map[string]interface{})
		if !ok {
			return in1
		}
		for k, v2 := range in2 {
			if v1, ok := in1[k]; ok {
				in1[k] = merge(v1, v2)
			} else {
				in1[k] = v2
			}
		}
	case nil:
		in2, ok := in2.(map[string]interface{})
		if ok {
			return in2
		}
	}
	return in1
}

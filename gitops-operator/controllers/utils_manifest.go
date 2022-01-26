package controllers

import (
	"bytes"
	"fmt"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"path/filepath"
)

func parseKeptnManifests(dir string, basedir string) (KeptnManifests, error) {
	repoPath := filepath.Join(dir, basedir)
	config := KeptnManifests{}

	files, err := ioutil.ReadDir(repoPath)
	if err != nil {
		return KeptnManifests{}, err
	}

	for _, file := range files {
		filetype := filepath.Ext(repoPath + file.Name())
		if filetype != ".yaml" && filetype != ".yml" {
			continue
		}

		yamlFile, err := ioutil.ReadFile(filepath.Join(repoPath, file.Name()))

		splitInput, err := SplitYAML(yamlFile)
		if err != nil {
			return KeptnManifests{}, fmt.Errorf("could not split yaml: %w", err)
		}

		scheme := runtime.NewScheme()
		err = apps.AddToScheme(scheme)
		if err != nil {
			return KeptnManifests{}, err
		}

		err = core.AddToScheme(scheme)
		if err != nil {
			return KeptnManifests{}, err
		}

		err = keptnv1.AddToScheme(scheme)
		if err != nil {
			return KeptnManifests{}, err
		}

		factory := serializer.NewCodecFactory(scheme)
		decoder := factory.UniversalDeserializer()

		objs := make([]interface{}, 0)

		for _, input := range splitInput {
			obj, _, err := decoder.Decode([]byte(input), nil, nil)
			if err != nil {
				fmt.Println("Could not parse file " + file.Name())
				fmt.Println(err)
				continue
			}
			objs = append(objs, obj)
		}

		for _, obj := range objs {
			switch obj.(type) {
			case *keptnv1.KeptnServiceDeployment:
				servicedeployment := obj.(*keptnv1.KeptnServiceDeployment)
				config.servicedeployments = append(config.servicedeployments, *servicedeployment)
			case *keptnv1.KeptnProject:
				project := obj.(*keptnv1.KeptnProject)
				config.projects = append(config.projects, *project)
			case *keptnv1.KeptnService:
				service := obj.(*keptnv1.KeptnService)
				config.services = append(config.services, *service)
			case *keptnv1.KeptnStage:
				stage := obj.(*keptnv1.KeptnStage)
				config.stages = append(config.stages, *stage)
			case *keptnv1.KeptnSequence:
				sequence := obj.(*keptnv1.KeptnSequence)
				config.sequences = append(config.sequences, *sequence)
			case *keptnv1.KeptnSequenceExecution:
				sequenceexecution := obj.(*keptnv1.KeptnSequenceExecution)
				config.execution = append(config.execution, *sequenceexecution)
			case *keptnv1.KeptnScheduledExec:
				scheduledexecution := obj.(*keptnv1.KeptnScheduledExec)
				config.scheduledexec = append(config.scheduledexec, *scheduledexecution)
			}

		}
	}
	return config, nil
}

func SplitYAML(resources []byte) ([][]byte, error) {

	dec := yaml.NewDecoder(bytes.NewReader(resources))

	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}

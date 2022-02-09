package utils

import (
	"context"
	"fmt"
	keptnv1 "github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator/api/v1"
	"github.com/keptn/go-utils/pkg/api/models"
	apiutils "github.com/keptn/go-utils/pkg/api/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FilterProjects returns an array of projects with the specified name
func FilterProjects(projects []*models.Project, projectName string) []*models.Project {
	filteredProjects := make([]*models.Project, 0)
	for _, project := range projects {
		if projectName == "" || projectName == project.ProjectName {
			filteredProjects = append(filteredProjects, project)
		}
	}
	return filteredProjects
}

// FilterServices returns an array of services with the specified name
func FilterServices(services []*models.Service, serviceName string) []*models.Service {
	filteredServices := make([]*models.Service, 0)
	for _, service := range services {
		if serviceName == "" || serviceName == service.ServiceName {
			filteredServices = append(filteredServices, service)
		}
	}
	return filteredServices
}

// GetKeptnCPToken returns the Keptn API Token in a Namespace
func GetKeptnCPToken(ctx context.Context, client client.Client, namespace string) (string, error) {
	keptnToken := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Name: "keptn-api-token", Namespace: namespace}, keptnToken)
	if err != nil {
		return "", fmt.Errorf("could not fetch keptn token: %w", err)
	}
	return string(keptnToken.Data["keptn-api-token"]), nil
}

// GetKeptnInstance returns the Keptn CP Instance Information in a Namespace
func GetKeptnInstance(ctx context.Context, client client.Client, namespace string) (keptnv1.KeptnInstance, string, error) {
	keptnInstance := keptnv1.KeptnInstance{}
	err := client.Get(ctx, types.NamespacedName{Name: "default", Namespace: namespace}, &keptnInstance)
	if err != nil {
		return keptnv1.KeptnInstance{}, "", fmt.Errorf("could not fetch keptn instance: %w", err)
	}

	token, err := DecryptSecret(keptnInstance.Status.CurrentToken)
	if err != nil {
		return keptnv1.KeptnInstance{}, "", err
	}

	return keptnInstance, token, nil
}

//CheckKeptnProjectExists queries the keptn api if a project exists
func CheckKeptnProjectExists(ctx context.Context, req ctrl.Request, clt client.Client, project string) (bool, error) {

	instance, token, err := GetKeptnInstance(ctx, clt, req.Namespace)
	if err != nil {

	}
	projectsHandler := apiutils.NewAuthenticatedProjectHandler(instance.Spec.APIUrl, token, instance.Status.AuthHeader, nil, instance.Status.Scheme)

	projects, err := projectsHandler.GetAllProjects()
	if err != nil {
		return false, err
	}

	filteredProjects := FilterProjects(projects, project)
	if len(filteredProjects) == 0 {
		if project != "" {
			return false, fmt.Errorf("no project"+project+"found: %w", err)
		}
		return false, err
	}
	return true, nil
}

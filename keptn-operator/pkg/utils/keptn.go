package utils

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/keptn/go-utils/pkg/api/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FilterProjects(projects []*models.Project, projectName string) []*models.Project {
	filteredProjects := make([]*models.Project, 0)
	for _, project := range projects {
		if projectName == "" || projectName == project.ProjectName {
			filteredProjects = append(filteredProjects, project)
		}
	}
	return filteredProjects
}

func FilterServices(services []*models.Service, serviceName string) []*models.Service {
	filteredServices := make([]*models.Service, 0)
	for _, service := range services {
		if serviceName == "" || serviceName == service.ServiceName {
			filteredServices = append(filteredServices, service)
		}
	}
	return filteredServices
}

func GetKeptnToken(client client.Client, logger logr.Logger, ctx context.Context, namespace string) string {
	keptnToken := &corev1.Secret{}
	err := client.Get(ctx, types.NamespacedName{Name: "keptn-api-token", Namespace: namespace}, keptnToken)
	if err != nil {
		logger.Info("Could not fetch KeptnToken")
	}
	return string(keptnToken.Data["keptn-api-token"])
}

package utils

import (
	"github.com/keptn/go-utils/pkg/api/models"
	"reflect"
	"testing"
)

func TestFilterProjects(t *testing.T) {
	type args struct {
		projects    []*models.Project
		projectName string
	}
	tests := []struct {
		name string
		args args
		want []*models.Project
	}{
		{
			name: "filter_found",
			args: args{
				projects: []*models.Project{
					{
						ProjectName: "project1",
					},
					{
						ProjectName: "project2",
					},
				},
				projectName: "project1",
			},
			want: []*models.Project{
				{
					ProjectName: "project1",
				},
			},
		},
		{
			name: "filter_not_found",
			args: args{
				projects: []*models.Project{
					{
						ProjectName: "project1",
					},
					{
						ProjectName: "project2",
					},
				},
				projectName: "project3",
			},
			want: []*models.Project{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterProjects(tt.args.projects, tt.args.projectName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterProjects() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterServices(t *testing.T) {
	type args struct {
		services    []*models.Service
		serviceName string
	}
	tests := []struct {
		name string
		args args
		want []*models.Service
	}{
		{
			name: "service_found",
			args: args{
				services: []*models.Service{
					{
						ServiceName: "service1",
					},
					{
						ServiceName: "service2",
					},
				},
				serviceName: "service1",
			},
			want: []*models.Service{
				{
					ServiceName: "service1",
				},
			},
		},
		{
			name: "service_found",
			args: args{
				services: []*models.Service{
					{
						ServiceName: "service1",
					},
					{
						ServiceName: "service2",
					},
				},
				serviceName: "service3",
			},
			want: []*models.Service{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FilterServices(tt.args.services, tt.args.serviceName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterServices() = %v, want %v", got, tt.want)
			}
		})
	}
}

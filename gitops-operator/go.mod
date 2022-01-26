module github.com/keptn-sandbox/keptn-gitops-operator/gitops-operator

go 1.16

require (
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-logr/logr v0.4.0
	github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator v0.0.0-00010101000000-000000000000
	github.com/spf13/afero v1.2.2
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
	k8s.io/api v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
	sigs.k8s.io/controller-runtime v0.10.0
)

replace github.com/keptn-sandbox/keptn-gitops-operator/keptn-operator => ../keptn-operator

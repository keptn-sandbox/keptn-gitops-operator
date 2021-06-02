module keptn-operator

go 1.13

require (
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.11.0
	github.com/onsi/gomega v1.8.1
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)

replace github.com/go-git/go-git/v5 => github.com/yeahservice/go-git/v5 v5.4.2-aws-patch

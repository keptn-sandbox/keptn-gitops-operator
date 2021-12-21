module github.com/keptn-sandbox/keptn-git-toolbox/promotion-service

go 1.16

require (
	github.com/cloudevents/sdk-go/v2 v2.3.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/keptn/go-utils v0.8.3
	github.com/keptn/kubernetes-utils v0.8.1
	github.com/mitchellh/mapstructure v1.2.2 // indirect
	github.com/onsi/ginkgo v1.12.0 // indirect
	github.com/onsi/gomega v1.9.0 // indirect
	github.com/spf13/afero v1.2.2
	github.com/stretchr/testify v1.7.0
	golang.org/x/tools v0.0.0-20200815165600-90abf76919f3 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776
	gotest.tools v2.2.0+incompatible
	helm.sh/helm/v3 v3.5.1
	k8s.io/apimachinery v0.20.4
)

replace github.com/go-git/go-git/v5 => github.com/yeahservice/go-git/v5 v5.4.2-aws-patch

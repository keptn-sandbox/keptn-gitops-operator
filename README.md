# Keptn GitOps Operators
The operators in this repository make keptn configurable via Custom Resources and via git.

## Compatibility Matrix

| Author           | Keptn Version | [Keptn GitOps Operator Images](https://hub.docker.com/r/checkelmann/gitlab-service/tags) |
|:-----------------|:---------------------:|:----------------------------------------------------------------------------------------:|
| @thschue         |      0.11.x         |    keptnsandbox/gitops-operator:0.1.0-dev <br> keptnsandbox/keptn-operator:0.1.0-dev     |


## Prerequisites
* In order to be able to create and delete stages, the keptn operator depends on a patched version of the configuration-service and the shipyard controller

## Installation
The operators and the promotion service, which is used to compose the files in the upstream repository are installed via helm. Following, the steps needed for deploying the operators are described.

### Prepare Keys for encryption of secrets
* Download Secrets helper from [Releases](https://github.com/keptn-sandbox/keptn-gitops-operator/releases)
* Create a keypair: ` ./keptn-gitops-secrets-(version) generate-keys -f <prefix>`
* Keep this secrets in a safe place, the public key will be needed for encrypting secrets, the private key to decrypt them on the server-side

### Prepare environment variables
Following, a few parameters have to be set upfront:
* `API_HOSTNAME` describes the hostname of the keptn/cloud automation instance (e.g. my-hostname.keptn.sh)
* `API_TOKEN` describes the Token of the keptn/cloud automation instance 
* `RSA_PRIVATE_KEY` represents the private key you created before
* `GITOPS_VERSION` specifies the Version you want to install (see [Releases](https://github.com/keptn-sandbox/keptn-gitops-operator/releases))

```shell
export API_HOSTNAME="<hostname>"
export API_TOKEN="<api-token>"
export RSA_PRIVATE_KEY="<private-key>"
export GITOPS_VERSION="0.1.0-pre.5"
```

### Install Custom Resource Definitions / Create Namespace
```
kubectl create namespace keptn
kubectl apply -f https://github.com/keptn-sandbox/keptn-gitops-operator/releases/download/${GITOPS_VERSION}/keptn-operator_crd.yaml
kubectl apply -f https://github.com/keptn-sandbox/keptn-gitops-operator/releases/download/${GITOPS_VERSION}/gitops-operator_crd.yaml
```

### Install Helm Chart
```
helm upgrade --install --atomic -n keptn keptn-gitops \
  https://github.com/keptn-sandbox/keptn-gitops-operator/releases/download/${GITOPS_VERSION}/keptn-gitops-${GITOPS_VERSION}.tgz \
  --set global.rsaSecret.privateBase64="${RSA_PRIVATE_KEY}" \
  --set promotion-service.remoteControlPlane.enabled=true \
  --set promotion-service.remoteControlPlane.api.protocol="https" \
  --set promotion-service.remoteControlPlane.api.hostname="${API_HOSTNAME}" --set promotion-service.remoteControlPlane.api.token="${API_TOKEN}"
```

## Keptn Operator
The operator introduces a set of custom resources to make keptn configurable via Kubernetes CRs.

### Custom Resources
|          Kind          |                    Purpose                    |                                Sample                                |
|:----------------------:|:---------------------------------------------:|:--------------------------------------------------------------------:|
|     KeptnInstance      |          Configure a Keptn Instance           |          [./samples/instance.yaml](./samples/instance.yaml)          |
|      KeptnProject      |           Configure a Keptn Project           |           [./samples/project.yaml](./samples/project.yaml)           |
|      KeptnService      |           Configure a Keptn Service           |           [./samples/service.yaml](./samples/service.yaml)           |
|     KeptnSequence      | Define a Keptn Sequence to be used in a Stage |         [./samples/sequence.yaml](./samples/sequences.yaml)          |
|       KeptnStage       |             Define a Keptn Stage              |             [./samples/stage.yaml](./samples/stage.yaml)             |
| KeptnServiceDeployment |  Specifies the deployed version of a service  | [./samples/servicedeployment.yaml](./samples/servicedeployment.yaml) |

### Usage:
* Create an empty upstream repository
* Create a KeptnInstance Custom Resource according to the [sample](./samples/instance.yaml). You can specify the secret to your secret either in clear text or RSA as an RSA encrypted string (prefix this with rsa:)
* Create a KeptnProject Custom Resource according to the [sample](./samples/project.yaml). You can specify the secret to your secret either in clear text or RSA as an RSA encrypted string (prefix this with rsa:)
* Create your keptn services according to the [sample](./samples/service.yaml). Ensure that you added the correct project.
* Create stages, and sequences. Ensure that you created the sequences you are referring to in the stage custom resources
* Define a service deployment to deploy the service

## GitOps Operator
The operator looks for configuration in a git repository, applies Keptn Custom Resources (see above) and pushes artifacts to the Keptn Upstream Repository.

### Custom Resources
|     Kind      |                         Purpose                          |                                Sample                                |
|:-------------:|:--------------------------------------------------------:|:--------------------------------------------------------------------:|
| KeptnGitRepository  | Defines a Repository containing your Keptn Configuration |           [./samples/gitrepo.yaml](./samples/gitrepo.yaml)           |

### Usage:
* Create an empty upstream repository
* Create a KeptnGitRepository Custom Resource according to the [sample](./samples/gitrepo.yaml). You can specify the secret to your secret either in clear text or RSA as an RSA encrypted string (prefix this with rsa:)
* Add your keptn configuration in the `.keptn` directory of your repository


## Contributions
* If there are additional use-cases which might be covered, please raise a PR
* Every PR and other contributions are welcome
* If you have other questions, or ideas, just reach out via slack

## Known Issues
* Currently Services are created regardless if they exist or not, this leads to many "create" events shown in the keptn-bridge.
* When a branch changes, all services in this branch are deployed (which might not necessarily end up in a redeployment)
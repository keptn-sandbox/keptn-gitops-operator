# Keptn GitOps Operators
The operators in this repository make keptn configurable via Custom Resources and via git.

## Compatibility Matrix

| Author           | Keptn Version | [Keptn GitOps Operator Images](https://hub.docker.com/r/checkelmann/gitlab-service/tags) |
|:-----------------|:---------------------:|:----------------------------------------------------------------------------------------:|
| @thschue         |      0.11.x         |    keptnsandbox/gitops-operator:0.1.0-dev <br> keptnsandbox/keptn-operator:0.1.0-dev     |


## Prerequisites
* In order to be able to create and delete stages, the keptn operator depends on a patched version of the configuration-service and the shipyard controller

## Installation
```shell
helm install <TBD>
```

## Keptn Operator
The operator introduces a set of custom resources to make keptn configurable via Kubernetes CRs.

### Custom Resources
|     Kind      |                    Purpose                    |                                Sample                                |
|:-------------:|:---------------------------------------------:|:--------------------------------------------------------------------:|
| KeptnProject  |           Configure a Keptn Project           |           [./samples/project.yaml](./samples/project.yaml)           |
| KeptnService  |           Configure a Keptn Service           |           [./samples/service.yaml](./samples/service.yaml)           |
| KeptnSequence | Define a Keptn Sequence to be used in a Stage |          [./samples/sequence.yaml](./samples/sequence.yaml)          |
| KeptnStage   |  Define a Keptn Stage |             [./samples/stage.yaml](./samples/stage.yaml)             |
| KeptnSequenceExecution | Triggers a Keptn Sequence Execution | [./samples/sequenceexecution.yaml](./samples/sequenceexecution.yaml) |

### Usage:
* Create an empty upstream repository
* Create a KeptnProject Custom Resource according to the [sample](./samples/project.yaml). You can specify the secret to your secret either in clear text or RSA as an RSA encrypted string (prefix this with rsa:)
* Create your keptn services according to the [sample](./samples/service.yaml). Ensure that you added the correct project.
* Create stages, and sequences. Ensure that you created the sequences you are referring to in the stage custom resources
* Define a sequence execution to trigger a keptn event

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
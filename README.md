# keptn-gitops-operator

> **DISCLAIMER**: This project is a proof-of-concept and was developed and is not complete and not supported. The CRDs as well as the backing mechanisms may change over time. Please use with care (currently it didn't break anything)!


## Idea
This operator makes keptn (almost) configurable by git. It introduces Custom Resource Definitions for KeptnProjects and KeptnServices.

If a Keptn Project CR is available, the operator tries to find the projects GitHub secret (as stored by keptn) and looks for a `.keptn/config.yaml` file in the main branch of the repository.

## Installation
To run this from your machine and you already installed kubebuilder, use:
```
make install
make run
```

Otherwise (unless the installation mechanism is ready):

**A Helm Chart will be created soon**

Assign a lot of permissions to the default user (will be fixed when using helm):
```
kubectl create clusterrolebinding sa-admin --clusterrole=cluster-admin --serviceaccount=<namespace>:default
```

Install Custom Resource Definitions:
```
make install
```

Run the operator:
```
kubectl run keptn-operator --image=keptncontrib/keptn-gitops-operator:0.0.1-dev-init --image-pull-policy='Always'
```

## Prerequisites

* The project is already added to keptn and a git upstream configured (see https://keptn.sh/docs/0.8.x/manage/git_upstream/)
* The operator is installed

## Usage:
To use the gitops operator you need knowledge about the keptn git repository (e.g. where keptn expects the helm charts and a simple configuration file (.keptn/config.yaml in the main branch) which might look as follows:

```
metadata:
  initbranch: "dev"

services:
- name: "carts"
  triggerevent: "sh.keptn.event.integration.artifact-delivery.triggered"
```

The branch specified by `initbranch` will be watched for changes. If there are changes on this branch, the deployment will be triggered using the event specified under `services[*].name.triggerevent`. The services specified here will be created (and deleted) in keptn.

After checking in this file, a Custom Resource for the corresponding Keptn Project should be created:

Example:
```
apiVersion: keptn.operator.keptn.sh/v1
kind: KeptnProject
metadata:
  name: my-keptn-project
spec:
  project: my-keptn-project
```

After applying this, the services defined in the config file will be managed in a GitOps way and the events specified will be triggered on a git push to the keptn configuration repository.

## Contributions
* If there are additional use-cases which might be covered, please raise a PR
* Every PR and other contributions are welcome
* If you have other questions, or ideas, just reach out via slack

## Known Issues
* Currently Services are created regardless if they exist or not, this leads to many "create" events shown in the keptn-bridge.
* When a branch changes, all services in this branch are deployed (which might not necessarily end up in a redeployment)

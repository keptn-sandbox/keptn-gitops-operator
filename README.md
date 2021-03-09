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

Assign a lot of permissions to the default user (will be fixed when using helm):
```
kubectl create clusterrolebinding sa-admin --clusterrole=cluster-admin --serviceaccount=<namespace>:default
```

Install Custom Resource Definitions:
```
kubectl cu
```

Run the operator:
```
kubectl run keptn-operator --image=keptncontrib/keptn-gitops-operator:0.0.1-dev-init --image-pull-policy='Always'
```

## Prerequisites



< Documentation in progress ...>
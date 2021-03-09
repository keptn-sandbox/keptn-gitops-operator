# keptn-gitops-operator

This is in a very early PoC stage, so please use with care and not in production

## Idea
This operator makes keptn (almost) configurable by git. It introduces Custom Resource Definitions for KeptnProjects and KeptnServices.

If a Keptn Project CR is available, the operator tries to find the projects GitHub secret (as stored by keptn) and looks for a `.keptn/config.yaml` file in the main branch of the repository.
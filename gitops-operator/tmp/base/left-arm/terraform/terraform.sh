#!/usr/bin/env sh

set -e

export TF_WORKING_DIR=/keptn/terraform
export GOOGLE_APPLICATION_CREDENTIALS=${TF_WORKING_DIR}/account.json
echo "${GCP_SA}" > ${TF_WORKING_DIR}/account.json

export TF_DATA_DIR=/tmp/.terraform/main
terraform -chdir=${TF_WORKING_DIR}/main init -backend-config="bucket=${GCP_BUCKET}" -backend-config="prefix=terraform/state-${KEPTN_SERVICE}-${KEPTN_STAGE}"
terraform -chdir=${TF_WORKING_DIR}/main validate
terraform -chdir=${TF_WORKING_DIR}/main plan -var="gke_project=${GCP_PROJECT}" -var="gke_region=${GCP_REGION}" -out ${TF_WORKING_DIR}/main.tfplan

#!/bin/bash
API_HOSTNAME="{{ .ApiUrl}}"
API_TOKEN="{{ .ApiToken}}"
RSA_PRIVATE_KEY="{{ .EncPrivateKey }}"
GITOPS_VERSION="{{ .GitOpsVersion }}"

kubectl create namespace keptn
kubectl apply -f https://github.com/keptn-sandbox/keptn-gitops-operator/releases/download/${GITOPS_VERSION}/keptn-operator_crd.yaml
kubectl apply -f https://github.com/keptn-sandbox/keptn-gitops-operator/releases/download/${GITOPS_VERSION}/gitops-operator_crd.yaml

helm upgrade --install --atomic -n keptn keptn-gitops \
  https://github.com/keptn-sandbox/keptn-gitops-operator/releases/download/${GITOPS_VERSION}/keptn-gitops-${GITOPS_VERSION}.tgz \
  --set global.rsaSecret.privateBase64="${RSA_PRIVATE_KEY}" \
  --set promotion-service.remoteControlPlane.enabled=true \
  --set promotion-service.remoteControlPlane.api.protocol="https" \
  --set promotion-service.remoteControlPlane.api.hostname="${API_HOSTNAME}" --set promotion-service.remoteControlPlane.api.token="${API_TOKEN}"


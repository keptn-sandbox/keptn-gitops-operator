set -e

/kubectl apply -f /keptn/crossplane/.
/kubectl wait --for=condition=Ready cloudmemorystoreinstance/${KEPTN_STAGE}-cloudmemorystore-instance --timeout=600s
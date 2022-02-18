#!/bin/bash

SERVICE=$1

if [ -z "$SERVICE" ]; then
  echo "No Service set, exiting..."
  exit 1
fi

make manifests
mkdir crds/

cat config/crd/bases/* > crds/${SERVICE}_crd.yaml

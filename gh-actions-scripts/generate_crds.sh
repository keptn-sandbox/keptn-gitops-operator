#!/bin/bash

SERVICE=$1

if [ -z "$SERVICE" ]; then
  echo "No Service set, exiting..."
  exit 1
fi

make manifests
make tmp
mkdir crds/

sed -i
cat config/crd/bases/* > tmp/${SERVICE}_crd.yaml
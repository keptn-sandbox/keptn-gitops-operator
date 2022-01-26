#!/usr/bin/env sh

cd /keptn/k6

k6 run -e STAGE=${STAGE} -e SERVICE=${SERVICE} -e SUBPATH=images/${SERVICE} --duration 60s --vus 30 /keptn/k6/smoke.js
#!/bin/ash

currentVersion=$(/kubectl get cm infra-version -ojsonpath='{.data.infraVersion}' -n keptn)
expectedVersion=0.0.0

if [ -f /keptn/infraVersion.txt ]; then
  expectedVersion=$(cat /keptn/infraVersion.txt)
fi

if [ "$(printf '%s\n' "$expectedVersion" "$currentVersion" | sort -V | head -n1)" = "$expectedVersion" ]; then
       echo "Greater than or equal to ${expectedVersion}"
       exit 0
else
       echo "Less than ${expectedVersion}"
       exit 1
fi
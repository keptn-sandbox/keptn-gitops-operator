#!/bin/bash
# shellcheck disable=SC2181

VERSION=$1 # e.g., 0.7.2-next.0
APP_VERSION=$2 # e.g., 0.7.2-next.0+1234
IMAGE=$3 # e.g., job-executor-service
CHART_DIR=$4

if [ $# -ne 4 ]; then
  echo "Usage: $0 VERSION APP_VERSION IMAGE"
  exit
fi

if [ -z "$VERSION" ]; then
  echo "No Version set, exiting..."
  exit 1
fi

if [ -z "$APP_VERSION" ]; then
  echo "No Image Tag set, defaulting to version"
  APP_VERSION=$VERSION
fi

if [ -z "$CHART_DIR" ]; then
  echo "No chart directory set, exiting"
  exit 1
fi


# replace "appVersion: latest" with "appVersion: $VERSION" in all Chart.yaml files
find ./$CHART_DIR -name Chart.yaml -exec sed -i -- "s/appVersion: latest/appVersion: \"${APP_VERSION}\"/g" {} \;
find ./$CHART_DIR -name Chart.yaml -exec sed -i -- "s/version: latest/version: \"${VERSION}\"/g" {} \;
find ./$CHART_DIR -name values.yaml -exec sed -i -- "s/latest/\"${VERSION}\"/g" {} \;

mkdir installer/

# ####################
# HELM CHART
# ####################
BASE_PATH=.
CHARTS_PATH=${CHART_DIR}

helm package ${BASE_PATH}/${CHARTS_PATH} --app-version "$APP_VERSION" --version "$VERSION"
if [ $? -ne 0 ]; then
  echo "Error packaging installer, exiting..."
  exit 1
fi

mv "${IMAGE}-${VERSION}.tgz" "installer/${IMAGE}-${VERSION}.tgz"
#verify the chart
helm template "installer/${IMAGE}-${VERSION}.tgz" --dry-run > /dev/null

if [ $? -ne 0 ]; then
  echo "::error Helm Chart for ${IMAGE} has templating errors -exiting"
  exit 1
fi

echo "Generated files:"
echo " - installer/${IMAGE}-${VERSION}.tgz"
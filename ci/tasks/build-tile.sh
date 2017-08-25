#!/usr/bin/env bash
set -e
source stackdriver-tools/ci/tasks/utils.sh

release_name="stackdriver-tools"
semver=`cat version-semver/number`
tile_name="${tile_name:-stackdriver-nozzle}"
tile_label="${tile_label:-'Stackdriver Nozzle'}"

check_param "image_directory"

check_param "image_directory"

image_name=${release_name}-${semver}.tgz
image_path="https://storage.googleapis.com/bosh-gcp/beta/${image_directory}/${image_name}"
output_path=candidate/stackdriver-nozzle-${semver}.pivotal

# install dependencies
apk update
apk add ruby

pushd "stackdriver-tools"
	echo "Creating tile.yml"
	RELEASE_PATH="${image_path}" TILE_NAME="${tile_name}" TILE_LABEL="${tile_label}" erb tile.yml.erb > tile.yml
	echo "building tile"
	tile build ${semver}
popd

echo "${image_path}"

echo "exposing tile"
mv stackdriver-tools/product/*.pivotal ${output_path}

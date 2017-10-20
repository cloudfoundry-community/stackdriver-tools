#!/usr/bin/env bash
set -e
source stackdriver-tools/ci/tasks/utils.sh

check_param "tile_name"
check_param "tile_label"

release_name="stackdriver-tools"
semver=`cat version-semver/number`
tile_name="${tile_name:-stackdriver-nozzle}"
tile_label="${tile_label:-'Stackdriver Nozzle'}"

image_path="$PWD/stackdriver-tools-artifacts/${release_name}-${semver}.tgz"
output_path=candidate/stackdriver-nozzle-${semver}.pivotal

# install dependencies
apk update
apk add ruby

pushd "stackdriver-tools"
	echo "Creating tile.yml"
	RELEASE_PATH="${image_path}" TILE_NAME="${tile_name}" TILE_LABEL="${tile_label}" erb tile.yml.erb > tile.yml
	echo "========================================================================"
	cat tile.yml
	echo "========================================================================"
	echo "building tile"
	tile build ${semver}
popd

echo "${image_path}"

echo "exposing tile"
mv stackdriver-tools/product/*.pivotal ${output_path}
echo -n $(sha256sum $output_path | awk '{print $1}') > $output_path.sha256
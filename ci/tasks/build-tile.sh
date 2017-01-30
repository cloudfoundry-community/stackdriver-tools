#!/usr/bin/env bash
set -e
source stackdriver-tools/ci/tasks/utils.sh

release_name="stackdriver-tools"
semver=`cat version-semver/number`

image_name=${release_name}-${semver}.tgz
image_path="https://storage.googleapis.com/bosh-gcp/beta/stackdriver-tools/${image_name}"
output_path=candidate/stackdriver-nozzle-${semver}.pivotal

pushd "stackdriver-tools"
	echo "Creating tile.yml"
	RELEASE_PATH=${image_path} erb tile.yml.erb > tile.yml
	echo "building tile"
	tile build ${semver}
popd

echo "${image_path}"

echo "exposing tile"
mv stackdriver-tools/product/*.pivotal ${output_path}

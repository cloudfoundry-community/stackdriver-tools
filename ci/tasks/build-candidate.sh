#!/usr/bin/env bash

set -e

source stackdriver-tools/ci/tasks/utils.sh

release_name="stackdriver-tools"
semver=`cat version-semver/number`
image_path=/tmp/${release_name}-${semver}.tgz

pushd stackdriver-tools
  echo "Using BOSH CLI version..."
  bosh2 --version

  echo "Exposing release semver to stackdriver-nozzle"
  echo ${semver} > "src/stackdriver-nozzle/release"

  echo "Fetching blobs"
  bosh2 sync-blobs

  # Force create because we just created the file `src/stackdriver-nozzle/release`
  echo "Creating ${release_name} BOSH Release..."
  bosh2 create-release --name=${release_name} --version=${semver} --tarball=${image_path} --force --sha2
popd

echo -n $(sha256sum $image_path | awk '{print $1}') > $image_path.sha256

mv ${image_path} candidate/
mv ${image_path}.sha256 candidate/

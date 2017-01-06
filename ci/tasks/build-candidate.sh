#!/usr/bin/env bash

set -e

source stackdriver-tools/ci/tasks/utils.sh
source /etc/profile.d/chruby-with-ruby-2.1.2.sh

release_name="stackdriver-tools"
semver=`cat version-semver/number`

pushd stackdriver-tools
  echo "Using BOSH CLI version..."
  bosh version

  echo "Exposing release semver to stackdriver-nozzle"
  echo ${semver} > "src/stackdriver-nozzle/release"

  # Force create because we just created the file `src/stackdriver-nozzle/release`
  echo "Creating ${release_name} BOSH Release..."
  bosh create release --name ${release_name} --version ${semver} --with-tarball --force
popd

image_path=stackdriver-tools/dev_releases/${release_name}/${release_name}-${semver}.tgz
echo -n $(sha1sum $image_path | awk '{print $1}') > $image_path.sha1

mv ${image_path} candidate/
mv ${image_path}.sha1 candidate/

#! /bin/bash
#
# Copyright 2019 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script builds a BOSH release of stackdriver-tools and uses the tile
# generator to package it into a tile. It expects VERSION to be set to a valid
# semantic version, GOPATH to be set, and PATH to include ${GOPATH}/bin, which
# must contain the `bosh2`, `golint` and `ginkgo` binaries.
#
# It is meant to be run by the custom-tile.sh script within a container, but
# if all the dependencies are satisfied locally it will probably work fine.

echo "${VERSION}" > "src/stackdriver-nozzle/release"
export RELEASE_PATH="dev_releases/stackdriver-tools-custom_${VERSION}.tgz"
# Clean up old builds.
rm -fr .dev_builds/* dev_releases/*
bosh2 sync-blobs
bosh2 create-release --force \
  --name="stackdriver-tools" \
  --version "${VERSION}" \
  --tarball="${RELEASE_PATH}"
echo "Exiting with $?"

export TILE_NAME="stackdriver-nozzle-custom"
export TILE_LABEL="Stackdriver Nozzle (custom build)"
erb tile.yml.erb > tile.yml
tile build "${VERSION}"
echo "Exiting with $?"

TILE="product/${TILE_NAME}-${VERSION}.pivotal"
sha256sum "${PWD}/${TILE}"
echo done

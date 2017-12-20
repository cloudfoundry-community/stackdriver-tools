#! /bin/bash

# This script builds a BOSH release of stackdriver-tools and uses the tile
# generator to pacakge it into a tile. It expects VERSION to be set to a valid
# semantic version, GOPATH to be set, and PATH to include $GOPATH/bin, which
# must contain the `bosh2`, `golint` and `ginkgo` binaries.
#
# It is meant to be run by the custom-tile script within a container, but
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

export TILE_NAME="stackdriver-nozzle-custom"
export TILE_LABEL="Stackdriver Nozzle (custom build)"
erb tile.yml.erb > tile.yml
tile build "${VERSION}"
TILE="product/${TILE_NAME}-${VERSION}.pivotal"
sha256sum "${PWD}/${TILE}"

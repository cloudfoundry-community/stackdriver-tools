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

#
# This script can be used to build the current working tree into a custom tile
# using the cfplatformeng/tile-generator image from Docker Hub.
#
# Before using it to build the tile, you must create a local customization of
# the tile generator image that contains all the build dependencies with
# `scripts/custom-tile setup`.
#
# Then, running `scripts/custom-tile build` will copy the current working tree
# into a container based on this customized image and run the build script.
# This will create the tile in the root directory of the repo. Note that it
# will be owned by root:root, because Docker.
#
# When you are done building, running `scripts/custom-tile clean` will delete
# the custom image from your local Docker repo.

# Change to the root of the repo.
if ! git rev-parse; then
  echo "You need to run this script from within the stackdriver-tools git repo."
  exit 1
fi
cd "$(git rev-parse --show-toplevel)"

export VERSION="0.0.$(date +%s)-custom.$(git rev-parse --short HEAD)"
export TMP_GOPATH="/tmp/gopath"

usage() {
  echo "Usage: build-custom-tile setup|build|clean"
}

setup() {
  cat > Dockerfile <<__DOCKERFILE__
FROM cfplatformeng/tile-generator:latest

WORKDIR /tmp
ENV GOPATH="${TMP_GOPATH}" PATH="${PATH}:${TMP_GOPATH}/bin"
COPY scripts/build-docker-env.sh .
RUN ./build-docker-env.sh
__DOCKERFILE__
  docker build -t tile-generator . && rm Dockerfile
}

build() {
  cat > Dockerfile <<__DOCKERFILE__
FROM tile-generator:latest

WORKDIR /tmp/stackdriver-tools
ENV VERSION="${VERSION}" GOPATH="${TMP_GOPATH}" PATH="${PATH}:${TMP_GOPATH}/bin"
COPY . .
RUN scripts/build-custom-tile-docker.sh
__DOCKERFILE__

  # remove previous custom tiles
  rm -f stackdriver-nozzle-custom*

  # remove previous tile containers and image (safely)
  docker ps -a | awk '{ if ($2 == "tile") print $1 }' | xargs -r docker rm
  docker images | cut -d ' ' -f 1 | grep -x tile | xargs -r docker rmi

  docker build -t tile . && rm Dockerfile

  TILE_DIR=/tmp/stackdriver-tools/product/
  TILE=stackdriver-nozzle-custom-${VERSION}.pivotal

  docker run -v "${PWD}:/mnt" tile sh -c "cp ${TILE_DIR}/${TILE} /mnt && chown ${UID} /mnt/${TILE}"
}

clean() {
  docker rmi tile-generator:latest
}

case "$1" in
"setup")
  # custom-tile setup creates a local customization of the
  # cloud foundry tile generator docker image with all the
  # dependencies needed to build the Stackdriver Nozzle.
  setup
  ;;
"build")
  # custom-tile build uses the customized tile-generator image
  # to build the stackdriver-tools BOSH release and the custom tile.
  IMAGE="$(docker images -q tile-generator:latest)"
  if test -z "${IMAGE}"; then
    echo "Custom tile generator image not found, running setup."
    setup
  fi
  build
  ;;
"clean")
  # custom-tile clean deletes the customized tile-generator image.
  clean
  ;;
*)
  usage
  exit 1
  ;;
esac

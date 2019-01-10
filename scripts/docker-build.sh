#!/usr/bin/env bash
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

usage() {
  echo "Usage: docker-build.sh bosh-release|tile|clean"
}

bosh-release() {
  docker build -t stackdriver-tools-tile-generator ./scripts
  docker run --user $(id -u) --rm -v $(pwd):/code --workdir /code stackdriver-tools-tile-generator make clean bosh-release
}

tile() {
  docker build -t stackdriver-tools-tile-generator ./scripts
  pwd
  docker run --user $(id -u) --rm -v $(pwd):/code --workdir /code stackdriver-tools-tile-generator make clean tile
}

clean() {
  docker rmi stackdriver-tools-tile-generator:latest
}

case "$1" in
"bosh-release")
  bosh-release
  ;;
"tile")
  tile
  ;;
"clean")
  clean
  ;;
*)
  usage
  exit 1
  ;;
esac

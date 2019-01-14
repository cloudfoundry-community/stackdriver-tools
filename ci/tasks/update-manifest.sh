#!/usr/bin/env sh
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

set -e # exit immediately if a simple command exits with a non-zero status
set -u # report the usage of uninitialized variables

mkdir -p edited-manifest
yq w cf-deployment-source/cf-deployment.yml stemcells[0].version "$(cat gcp-xenial-stemcells/version)" > edited-manifest/cf-deployment.yml

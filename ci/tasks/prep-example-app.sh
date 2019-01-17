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

set -e # exit immediately if a simple command exits with a non-zero status
set -u # report the usage of uninitialized variables

cp -R stackdriver-tools-source/* prepped_source/
echo ${GOOGLE_APPLICATION_CREDENTIALS} > prepped_source/examples/cf-stackdriver-example/credentials.json
cd stackdriver-tools-source

cat <<EOF > ../prepped_source/examples/cf-stackdriver-example/source-context.json

{
  "git": {
    "revisionId": "$(git rev-parse HEAD)",
    "url": "${STACKDRIVER_TOOLS_SOURCE_URI}"
  }
}
EOF

cd ../prepped_source/examples/cf-stackdriver-example/

# Update the debug agent binary
# This ensures it's compiled with the same version of go as the example app.
go get -u cloud.google.com/go/cmd/go-cloud-debug-agent
mv ${GOPATH}/bin/go-cloud-debug-agent ./go-cloud-debug

../../../stackdriver-tools-source-ci/ci/setup-gopath.sh go build -o ./cf-stackdriver-example -gcflags=all='-N -l'

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
# Dockerfile for building BOSH release and PCF Tile

# TODO mattysweeps: https://github.com/cloudfoundry-community/stackdriver-tools/issues/223
FROM cfplatformeng/tile-generator:latest

# musl-dev needed for ld: https://bugs.alpinelinux.org/issues/6628
RUN apk --no-cache add go ruby musl-dev coreutils make

RUN go get github.com/onsi/ginkgo && \
    go install github.com/onsi/ginkgo/... && \
    go get github.com/golang/lint/golint

CMD make clean bosh-release

# Set HOME dir to source code dir, so BOSH can create folders when running as non-root user
WORKDIR /code
ENV HOME=/code

#! /bin/sh

# This script is run from inside the tile-generator container.  It installs the
# dependencies necessary for building the nozzle.  Most of the contents have
# been gratuitously stolen from the CI configs + Dockerfile.

mkdir -p "${GOPATH}/bin"

apk update
# musl-dev needed for ld: https://bugs.alpinelinux.org/issues/6628
# coreutils needed for sha256sum and sha1sum
apk add go ruby musl-dev coreutils

go get github.com/onsi/ginkgo
go install github.com/onsi/ginkgo/...
go get github.com/golang/lint/golint

## Install Bosh 2 CLI
BOSH2_VERSION=2.0.48
BOSH2_SHA1="c807f1938494f4280d65ebbdc863eda3f883d72e"

wget -q -c "https://s3.amazonaws.com/bosh-cli-artifacts/bosh-cli-${BOSH2_VERSION}-linux-amd64"
echo "${BOSH2_SHA1}	bosh-cli-${BOSH2_VERSION}-linux-amd64" > "bosh2_${BOSH2_VERSION}_SHA1SUM"
if ! sha1sum -cw --status "bosh2_${BOSH2_VERSION}_SHA1SUM"; then exit 1; fi
mv "bosh-cli-${BOSH2_VERSION}-linux-amd64" "gopath/bin/bosh2"
chmod a+x "gopath/bin/bosh2"

go version # Go in Alpine v3.6 is 1.8.4
ginkgo version
ruby --version
bosh2 --version

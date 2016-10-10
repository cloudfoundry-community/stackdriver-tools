#!/usr/bin/env bash

set -e

export GOPATH=${PWD}/gcp-tools-release
export PATH=${GOPATH}/bin:$PATH

cd ${PWD}/gcp-tools-release/src/stackdriver-nozzle
make test

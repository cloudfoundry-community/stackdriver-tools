#!/usr/bin/env bash

set -e

# Create a workspace for a GOPATh
gopath_prefix=/tmp/src/github.com/cloudfoundry-community
mkdir -p ${gopath_prefix}

# Link to the source repo
ln -s ${PWD}/stackdriver-tools ${gopath_prefix}/

# Configure GOPATH
export GOPATH=/tmp
export PATH=${GOPATH}/bin:$PATH

# Run tests
cd ${gopath_prefix}/gcp-tools-release/src/stackdriver-nozzle
make test

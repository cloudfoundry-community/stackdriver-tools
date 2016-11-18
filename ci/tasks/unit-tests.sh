#!/usr/bin/env bash

set -e

# Create a workspace for a GOPATh
gopath_prefix=/tmp/src/github.com/stackdriver-tools
mkdir -p ${gopath_prefix}

# Link to the source repo
ln -s ${PWD}/stackdriver-tools ${gopath_prefix}/stackdriver-tools

# Configure GOPATH
export GOPATH=/tmp
export PATH=${GOPATH}/bin:$PATH

# Run tests
cd ${gopath_prefix}/stackdriver-tools/src/stackdriver-nozzle
make test

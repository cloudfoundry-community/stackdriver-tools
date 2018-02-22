#!/usr/bin/env bash

set -e

source stackdriver-tools/ci/tasks/utils.sh

mv stackdriver-tools-artifacts/*.tgz candidate/latest.tgz
mv stackdriver-tools-artifacts-sha256/*.tgz.sha256 candidate/latest.tgz.sha256
mv stackdriver-nozzle-tile/*.pivotal candidate/latest.pivotal

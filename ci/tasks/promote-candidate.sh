#!/usr/bin/env bash

set -e

source stackdriver-tools/ci/tasks/utils.sh

mv stackdriver-tools-artifacts/*.tgz candidate/latest.tgz
mv stackdriver-tools-artifacts-sha1/*.tgz.sha1 candidate/latest.tgz.sha1
mv stackdriver-nozzle-tile/*.pivotal candidate/latest.pivotal

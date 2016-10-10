#!/usr/bin/env bash

set -e

source gcp-tools-release/ci/tasks/utils.sh
source /etc/profile.d/chruby-with-ruby-2.1.2.sh

check_param bosh_director_address
check_param bosh_user
check_param bosh_password
check_param cf_deployment_name

echo "Using BOSH CLI version..."
bosh version

echo "Targeting BOSH director..."
bosh -n target ${bosh_director_address}
bosh login ${bosh_user} ${bosh_password}

echo "Uploading nozzle release..."
bosh upload release gcp-tools-release-artifacts/*.tgz

echo "Downloading existing Cloud Foundry manifest"
bosh download manifest ${cf_deployment_name} > cloudfoundry.yml
bosh deployment cloudfoundry.yml
bosh -n deploy

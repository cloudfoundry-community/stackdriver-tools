#!/usr/bin/env bash

set -ex

source stackdriver-tools/ci/tasks/utils.sh

# BOSH and CF config
check_param bosh_director_address
check_param bosh_user
check_param bosh_password
check_param bosh_ca_cert

# CF settings
check_param cf_api_url
check_param firehose_username
check_param firehose_password

# Google network settings
check_param google_zone
check_param google_region
check_param network
check_param private_subnetwork

# Google service account settings
check_param cf_service_account_json
check_param ssh_user
check_param ssh_key

semver=`cat version-semver/number`

echo "Configuring SSH"
echo -e "${ssh_key}" > /tmp/${ssh_user}.key
chmod 700 /tmp/${ssh_user}.key

echo "Configuring credentials"
echo "${cf_service_account_json}" > /tmp/service_account.json

echo "Connecting to SSH bastion..."
ssh -4 -D 5000 -fNC bosh@${ssh_bastion_address} -i /tmp/${ssh_user}.key -o StrictHostKeyChecking=no
export BOSH_ALL_PROXY=socks5://localhost:5000

echo "Using BOSH CLI version..."
bosh2 --version
export BOSH_CLIENT=${bosh_user}
export BOSH_CLIENT_SECRET=${bosh_password}
export BOSH_ENVIRONMENT=https://${bosh_director_address}:25555
export BOSH_CA_CERT=${bosh_ca_cert}

echo "Targeting BOSH director..."
bosh2 login -n
bosh2 env

echo "Uploading nozzle release..."
bosh2 upload-release stackdriver-tools-artifacts/*.tgz

pushd stackdriver-tools
echo "Updating cloud config"
bosh2 update-cloud-config -n manifests/cloud-config-gcp.yml \
          -v zone=${google_zone} \
          -v network=${network} \
          -v subnetwork=${private_subnetwork} \
          -v "tags=['stackdriver-nozzle']" \
          -v internal_cidr=10.0.0.0/16 \
          -v internal_gw=10.0.0.1 \
          -v "reserved=[10.0.0.1-10.0.0.10]"

bosh2 cloud-config

echo "Deploying nozzle release"
bosh2 deploy -n manifests/stackdriver-tools.yml \
            -d stackdriver-nozzle \
            -v firehose_endpoint=${cf_api_url} \
            -v firehose_username=${firehose_username} \
            -v firehose_password=${firehose_password} \
            -v skip_ssl=true \
            -v gcp_project_id=${cf_project_id} \
            --var-file gcp_service_account_json=/tmp/service_account.json

popd

# Move release and its SHA256
mv stackdriver-tools-artifacts/*.tgz candidate/latest.tgz
mv stackdriver-tools-artifacts-sha256/*.tgz.sha256 candidate/latest.tgz.sha256

#!/usr/bin/env bash

set -e

source gcp-tools-release/ci/tasks/utils.sh
source /etc/profile.d/chruby-with-ruby-2.1.2.sh

# BOSH and CF config
check_param bosh_director_address
check_param bosh_user
check_param bosh_password

# CF settings
check_param vip_ip
check_param nozzle_user
check_param nozzle_password

# Google network settings
check_param google_zone
check_param google_region
check_param network
check_param private_subnetwork

# Google service account settings
check_param cf_service_account
check_param service_account_key_json
check_param ssh_bastion_name
check_param ssh_user
check_param ssh_key

echo "Creating google json key..."
mkdir -p $HOME/.config/gcloud/
echo "${service_account_key_json}" > $HOME/.config/gcloud/application_default_credentials.json

echo "Configuring google account..."
gcloud auth activate-service-account --key-file $HOME/.config/gcloud/application_default_credentials.json
gcloud config set project ${project_id}
gcloud config set compute/region ${google_region}
gcloud config set compute/zone ${google_zone}

echo "Configuring SSH"
echo -e "${ssh_key}" > /tmp/${ssh_user}.key
chmod 700 /tmp/${ssh_user}.key

echo "Connecting to SSH bastion..."
ssh bosh@${ssh_bastion_address} -i /tmp/${ssh_user}.key -o StrictHostKeyChecking=no -L 25555:${bosh_director_address}:25555 -nNT &

echo "Using BOSH CLI version..."
bosh version

echo "Targeting BOSH director..."
bosh -n target localhost
bosh login ${bosh_user} ${bosh_password}
director_uuid=$(bosh status --uuid)

echo "Uploading nozzle release..."
bosh upload release gcp-tools-release-artifacts/*.tgz

nozzle_manifest_name=stackdriver-nozzle.yml
cat > ${nozzle_manifest_name} <<EOF
---

name: stackdriver-nozzle-ci
director_uuid: ${director_uuid}

releases:
- name: bosh-gcp-tools
  version: latest

jobs:
- name: stackdriver-nozzle
  instances: 1
  networks:
    - name: private
  resource_pool: common
  templates:
    - name: stackdriver-nozzle
      release: bosh-gcp-tools
    - name: google-fluentd
      release: bosh-gcp-tools
    - name: stackdriver-agent
      release: bosh-gcp-tools
  properties:
    firehose:
      endpoint: https://api.${vip_ip}.xip.io
      events:   LogMessage,Error,HttpStartStop,CounterEvent,ValueMetric,ContainerMetric
      username: ${nozzle_user}
      password: ${nozzle_password}
      skip_ssl: true
    nozzle:
      debug: true

compilation:
  workers: 6
  network: private
  reuse_compilation_vms: true
  cloud_properties:
    zone: ${google_zone}
    machine_type: n1-standard-8
    root_disk_size_gb: 100
    root_disk_type: pd-ssd
    preemptible: true
    service_account: ${cf_service_account}

resource_pools:
  - name: common
    network: private
    stemcell:
      name: bosh-google-kvm-ubuntu-trusty-go_agent
      version: latest
    cloud_properties:
      zone: ${google_zone}
      machine_type: n1-standard-4
      root_disk_size_gb: 20
      root_disk_type: pd-standard
      service_account: ${cf_service_account}
  - name: nozzle
    network: private
    stemcell:
      name: bosh-google-kvm-ubuntu-trusty-go_agent
      version: latest
    cloud_properties:
      zone: ${google_zone}
      machine_type: n1-standard-4
      root_disk_size_gb: 20
      root_disk_type: pd-standard
      service_account: ${cf_service_account}

networks:
  - name: private
    type: manual
    subnets:
    - range: 192.168.0.0/16
      reserved:
      - 192.168.0.0-192.168.1.15
      gateway: 192.168.0.1
      cloud_properties:
        zone: ${google_zone}
        network_name: ${network}
        subnetwork_name: ${private_subnetwork}
        ephemeral_external_ip: false
        tags:
          - stackdriver-nozzle-internal
          - internal
          - no-ip

update:
  canaries: 1
  max_in_flight: 1
  serial: false
  canary_watch_time: 1000-60000
  update_watch_time: 1000-60000

EOF

bosh deployment ${nozzle_manifest_name}
bosh -n deploy

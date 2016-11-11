#!/usr/bin/env bash

set -e

source stackdriver-tools/ci/tasks/utils.sh
source /etc/profile.d/chruby-with-ruby-2.1.2.sh

# BOSH and CF config
check_param project_id
check_param service_account_key_json
check_param bucket_name
check_param semver_key

echo "Creating google json key..."
mkdir -p $HOME/.config/gcloud/
echo "${service_account_key_json}" > $HOME/.config/gcloud/application_default_credentials.json

echo "Configuring google account..."
gcloud auth activate-service-account --key-file $HOME/.config/gcloud/application_default_credentials.json
gcloud config set project ${project_id}

echo "Making semver public"
gsutil acl ch -u allUsers:R gs://$bucket_name/$semver_key

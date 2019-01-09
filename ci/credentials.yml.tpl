#
# Copyright 2019 Google Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
---

# source code
stackdriver_tools_uri: https://github.com/cloudfoundry-community/stackdriver-tools.git
stackdriver_tools_branch: master
stackdriver_tools_ci_uri: https://github.com/cloudfoundry-community/stackdriver-tools.git
stackdriver_tools_ci_branch: master

# GCP account and credentials
gcp_project_id: replace-me
gcp_region: replace-me
service_account_key_json: replace-me

# BOSH configuration
domain_name: replace-me
bbl-state-bucket: replace-me
bbl_env_name: replace-me
load_balancer_cert: replace-me
load_balancer_key: replace-me

# Output resource configuration
resources_bucket_name: replace-me

# CloudFoundry configuration
cf_api_endpoint: replace-me-after-env-up
cf_username: replace-me-after-env-up
cf_password: replace-me-after-env-up
skip_ssl: true

# Stackdriver spinner configuration
spinner_org: replace-me
spinner_space: replace-me

# Example app configuration
example_app_org: replace-me
example_app_space: replace-me

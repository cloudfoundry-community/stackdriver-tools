---
# Google Cloud Storage
bucket_access_key: {{bucket_access_key}} # GCS interop access key
bucket_secret_key: {{bucket_secret_key}} # GCS interop secret key
bucket_name:       {{bucket_name}} # GCS bucket for semver storage

# Google service account
service_account: {{service_account}}
service_account_key_json: |
  {{service_account_key_json}}

# BOSH and Cloud Foundry
ssh_bastion_address:      {{ssh_bastion_name}} # Instance name of a VM with public SSH and access to the BOSH director
ssh_user:              {{ssh_user}} # The SSH user that can connecto to the bastion
ssh_key: {{ssh_key}} # The SSH key for ssh_user
bosh_director_address: {{bosh_director_address}} # IP address of BOSH director
bosh_user:             {{bosh_user}} # Bosh admin username
bosh_password:         {{bosh_password} # Bosh password
cf_deployment_name:    {{cf_deployment_name}} # Name of CF deployment to update

# CF settings
vip_ip:          {{replace_me}}
common_password: {{replace_me}}

# Google network settings
google_region:      {{replace_me}}
google_zone:        {{replace_me}}
network:            {{replace_me}}
public_subnetwork:  {{replace_me}}
private_subnetwork: {{replace_me}}

# Google service account settings
project_id:                  {{replace_me}}
nozzle_user:                 {{replace_me}}
nozzle_password:             {{replace_me}}
cf_project_id:               {{replace_me}}
cf_service_account_json:     {{replace_me}}

# Slack
slack-hook: {{slack webhook to post to our channel}} # see https://api.slack.com/incoming-webhooks

# GitHub
github_deployment_key_bosh_google_cpi_release: | # GitHub deployment key for release artifacts
github_pr_access_token: # An access token with repo:status access, used to test PRs

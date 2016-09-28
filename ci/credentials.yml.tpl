---
# Google Cloud Storage
bucket_access_key: {{bucket_access_key}} # GCS interop access key
bucket_secret_key: {{bucket_secret_key}} # GCS interop secret key
bucket_name:       {{bucket_name}} # GCS bucket for semver storage

# Google service account
service_account: {{service_account}}
service_account_key_json: |
  {{service_account_key_json}}

# GitHub
github_deployment_key_bosh_google_cpi_release: | # GitHub deployment key for release artifacts
github_pr_access_token: # An access token with repo:status access, used to test PRs

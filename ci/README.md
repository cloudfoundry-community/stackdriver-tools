# Deploy CI Infrastructure

1. Export your project ID, a prefix, and the pipeline (develop or prod):

  ```
  projectid=<REPLACE_WITH_YOUR_PROJECT_ID>
  prefix=stackdriver-tools
  pipeline=develop
  ```

1. Create a service account and key:

  ```
  gcloud iam service-accounts create ${pipeline}-${prefix}
  gcloud iam service-accounts keys create /tmp/${pipeline}-${prefix}.key.json \
      --iam-account ${pipeline}-${prefix}@${projectid}.iam.gserviceaccount.com
  ```

1. Grant the new service account editor access to your project:

  ```
  gcloud projects add-iam-policy-binding ${projectid} \
      --member serviceAccount:${pipeline}-${prefix}@${projectid}.iam.gserviceaccount.com \
      --role roles/storage.admin
  ```
1. Provision required infrastructure with `terraform` and the `main.tf` manifest in this directory:

  ```
  terraform apply -var projectid=${projectid} -var pipeline=${pipeline}
  ```

1. If you want release blobs to be downloadable without authentication, go to the [Storage browser](https://console.cloud.google.com/storage/browser/) console, choose **Edit object default permissions** for the bucket created by Terraform, and add a `publicRead` entry for the `allUsers` user.

1. Copy the credentials template file:

  ```
  cp credentials.yml.tpl credentials-${pipeline}.yml
  ```

1. Update your credentials file:

  ```
  sed -i "s/{{bucket_name}}/$(terraform state show google_storage_bucket.pipeline-artifacts | grep id | awk -F '= ' '{print $2}')/" credentials-${pipeline}.yml
  sed -i "s/{{service_account}}/${pipeline}-${prefix}@${projectid}.iam.gserviceaccount.com/" credentials-${pipeline}.yml
  sed -i "s|{{service_account_key_json}}|`cat /tmp/${pipeline}-${prefix}.key.json`|" credentials-${pipeline}.yml
  ``` 

1. Navigate to the [Storage section](https://console.cloud.google.com/storage/settings) of the Google Cloud console, choose the **Interoperability** section, and click **Create a new key**. You will need the **Access key** and **Secret** generated from this step.

1. Edit `credentials-*.yml` and set the `bucket_access_key` and `bucket_secret_key` with the values you just created.

1. Edit `credentials-*.yml` and replace ` {{service_account_key_json}}` with the contents of `cat /tmp/${pipeline}-${prefix}.key.json`

# Deploy Pipeline

1. Login to Concourse:

  ```
  fly --target lambr login \
    --team-name stackdriver-tools \
    --concourse-url https://your-concourse-name -k
  ```

1. Upload the pipeline:

  ```
fly -t google set-pipeline -p ${pipeline}-stackdriver-tools -c pipeline-${pipeline}.yml -l credentials-${pipeline}.yml
  ```
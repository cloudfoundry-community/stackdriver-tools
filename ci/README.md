# Deploy CI Infrastructure

Export your project ID, a prefix, and the pipeline (develop or prod):

  ```
  projectid=<REPLACE_WITH_YOUR_PROJECT_ID>
  prefix=stackdriver-tools
  pipeline=develop
  ```

Create a service account and key:

  ```
  gcloud iam service-accounts create ${pipeline}-${prefix}
  gcloud iam service-accounts keys create /tmp/${pipeline}-${prefix}.key.json \
      --iam-account ${pipeline}-${prefix}@${projectid}.iam.gserviceaccount.com
  ```

Grant the new service account editor access to your project:

  ```
  gcloud projects add-iam-policy-binding ${projectid} \
      --member serviceAccount:${pipeline}-${prefix}@${projectid}.iam.gserviceaccount.com \
      --role roles/storage.admin
  ```

Provision required infrastructure with `terraform` and the `main.tf` manifest in this directory:

  ```
  terraform apply -var projectid=${projectid} -var pipeline=${pipeline}
  ```

If you want release blobs to be downloadable without authentication, go to the [Storage browser](https://console.cloud.google.com/storage/browser/) console, choose **Edit object default permissions** for the bucket created by Terraform, and add a `publicRead` entry for the `allUsers` user.

Copy the credentials template file and update its values appropriately: (use the key generated above)

  ```
  cp credentials.yml.tpl credentials-${pipeline}.yml
  ```

# Deploy Pipeline

1. Login to Concourse:

  ```
  fly --target my_target login \
    --team-name my_team \
    --concourse-url https://your-concourse-name -k
  ```

1. Upload the pipeline:

  ```
fly -t my_team set-pipeline -p ${pipeline}-stackdriver-tools -c pipeline-${pipeline}.yml -l credentials-${pipeline}.yml
  ```
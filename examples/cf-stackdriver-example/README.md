# cf-stackdriver-example

A Cloud Foundry application written in Go that reports errors to [Stackdriver Error Reporting](https://cloud.google.com/error-reporting/) and can be debugged with the [Stackdriver Debugger](https://cloud.google.com/debugger/).

Based on the [Kubernetes Guestbook App](https://github.com/kubernetes/kubernetes/tree/master/examples/guestbook-go). This is ***not*** an official Google product.

## Prerequisites

- [Cloud Foundry](https://cloud.google.com/solutions/cloud-foundry-on-gcp) with the CLI signed in and targeting a Cloud Foundry you have push access to
- A Google project with the following APIs enabled:
  - [Stackdriver Error Reporting API](https://console.cloud.google.com/apis/api/clouderrorreporting.googleapis.com/overview)
  - [Stackdriver Debugger API](https://console.cloud.google.com/apis/api/clouddebugger.googleapis.com/overview)
  - [Cloud Datastore API](https://console.cloud.google.com/apis/api/datastore.googleapis.com/overview)
- To test the database, the Google project must have an existing App Engine project. [More info](https://cloud.google.com/datastore/docs/activate).
- [Go 1.11](https://golang.org/)
- [Cloud SDK](https://cloud.google.com/sdk/downloads)

## Deploying the application

The application requires the GOOGLE_PROJECT environment variable to start. The following snippet will create the app, set the variable, and re-stage the app at which time it should start.

```
cf push
export project_id=$(gcloud config get-value project 2>/dev/null)
cf set-env cf-stackdriver-example GOOGLE_PROJECT ${project_id}
cf restage cf-stackdriver-example
```

## Setting up IAM permissions

The following IAM roles are required for the service account of this application. If your Cloud Foundry is deployed on GCP with these default service account credentials then you can skip this step.
- roles/clouddebugger.agent
- roles/logging.logWriter
- roles/datastore.owner


Use the following command to create a service account with these roles:
```
export project_id=$(gcloud config get-value project 2>/dev/null)
export account_name=cf-stackdriver-example
export service_account_email=${account_name}@${project_id}.iam.gserviceaccount.com
gcloud iam service-accounts create ${account_name}
gcloud projects add-iam-policy-binding ${project_id} --member serviceAccount:${service_account_email} --role "roles/clouddebugger.agent"
gcloud projects add-iam-policy-binding ${project_id} --member serviceAccount:${service_account_email} --role "roles/logging.logWriter"
gcloud projects add-iam-policy-binding ${project_id} --member serviceAccount:${service_account_email} --role "roles/datastore.owner"
gcloud iam service-accounts keys create service_account.json \
    --iam-account ${service_account_email}
```

Pass these credentials to your application and deploy it:
```
cf set-env cf-stackdriver-example GOOGLE_APPLICATION_CREDENTIALS '/home/vcap/app/service_account.json'
./deploy
```

## Try it out

Navigate to your application's URL and attempt to submit a guestbook entry. You should see a new error in the [Stackdriver Errors](https://console.cloud.google.com/errors) console. Try setting a debug breakpoint in main.go:69 and see if you can spot the bug.

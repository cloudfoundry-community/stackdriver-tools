# Clearing Project Metric Descriptors

This procedure will delete all custom metric descriptors in your Stackdriver Monitoring project.
This results in historical data loss and the need to re-create dashboards.

## Prerequisites
- [Golang 1.9+](https://golang.org/doc/install)
- [BOSH-CLI 2.0.48+](https://bosh.io/docs/cli-v2.html#install) (if using the `stackdriver-tools` BOSH release)
- [Google Cloud SDK](https://cloud.google.com/sdk/downloads)

## 1. Authenticate with Google Cloud SDK
Authenticate with an account that has `roles/monitoring.admin` to the Stackdriver Monitoring project
and setup your application default credentials.

```bash
gcloud auth login
gcloud auth appliaction-default login
```

## 2. Fetch `clear-metrics-descriptors`
```bash
go get -d github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle
``` 

## 3. Stop all instance of `stackdriver-nozzle`
All copies of the `stackdriver-nozzle` need to be completely stopped before proceeding. If an instance
is left running then old metric descriptors will be recreated.

If you're using the Stackdriver Nozzle tile in Pivotal Operations Manager then follow [these instructions](https://docs.pivotal.io/pivotalcf/2-0/customizing/add-delete.html)
to delete the product and apply changes to your deployment.

If you're using the `stackdriver-tools` BOSH release, run: `bosh -d <your deployment> --stop stackdriver-nozle`

## 4. Clear Metric Descriptors
Enter the directory with the clear-metrics-descriptors tool:
```bash
cd $(go env GOPATH)/src/github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cmd/clear-metrics-descriptors
```

Fill in your GCP Project in the following command and execute the tool:
```bash
go run ./clear-metrics-descriptors.go --project-id "your GCP project, eg cf-prod-logs"
```

Your project should now be clear of all custom metric descriptors. You can proceed with upgrading the nozzle.

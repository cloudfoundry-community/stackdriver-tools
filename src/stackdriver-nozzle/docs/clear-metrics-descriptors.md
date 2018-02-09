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

## 2. Fetch and build `clear-metrics-descriptors
```bash
go get -d github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle
GOBIN=`pwd` go install $(go env GOPATH)/src/github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cmd/clear-metrics-descriptors.go
ls clear-metrics-descriptors
``` 

## 3. Stop all instance of `stackdriver-nozzle`
All copies of the `stackdriver-nozzle` need to be completely stopped before proceeding. If an instance
is left running then old metric descriptors will be recreated.

If you're using the Stackdriver Nozzle tile in Pivotal Operations Manager then follow [these instructions](https://docs.pivotal.io/pivotalcf/2-0/customizing/add-delete.html)
to delete the product and apply changes to your deployment.

If you're using the `stackdriver-tools` BOSH release, run: `bosh -d <your deployment> --stop stackdriver-nozle`

## 4. Clear Metric Descriptors
Export the follow environment variable to the name of your Stackdriver Monitoring project:
```bash
export GCP_PROJECT_ID=<Your GCP Project ID, eg cf-prod-monitoring-foo>
```

Execute the `clear-metrics-descriptors` program you compiled in step 2:
```bash
./clear-metrics-descriptors
```

Your project should now be clear of all custom metric descriptors. You can proceed with upgrading the nozzle.
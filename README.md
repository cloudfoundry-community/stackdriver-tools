# stackdriver-tools release for BOSH

This release provides Cloud Foundry and BOSH integration with Google Cloud
Platform's [Stackdriver Logging](https://cloud.google.com/logging/) and
[Monitoring](https://cloud.google.com/monitoring/).

Functionality is provided by 3 jobs in this release:

* A [nozzle][nozzle] job for forwarding [Cloud Foundry Firehose][firehose] data to Stackdriver
* A [Fluentd][fluentd] job for forwarding syslog and template logs to [Stackdriver Logging][logging]
* A [Stackdriver Monitoring Agent][monitoring-agent] job for sending VM health
  metrics to [Stackdriver Monitoring][monitoring]

[monitoring]: https://cloud.google.com/monitoring/
[fluentd]: http://www.fluentd.org/
[monitoring-agent]: https://cloud.google.com/monitoring/agent/
[logging]: https://cloud.google.com/logging/
[firehose]: https://docs.cloudfoundry.org/loggregator/architecture.html#firehose
[nozzle]: src/stackdriver-nozzle

## Project Status

The following is generally available:
- Stackdriver Host Monitoring Agent (`stackdriver-agent`) 
- Stackdriver Host Logging Agent (`google-fluentd`)
- Stackdriver Nozzle (`stackdriver-nozzle`)
  - Stackdriver Logging for Cloud Foundry Log Events (`LogMessage, Error, HttpStartStop`)
  - Stackdriver Monitoring for Cloud Foundry Metric Events (`ContainerMetric, ValueMetric, CounterEvent`)

The following is in beta:
- Stackdriver Nozzle
   - Stackdriver Logging for Cloud Foundry Metric Events (`ContainerMetric, ValueMetric, CounterEvent`)

The project was developed in partnership with Google and Pivotal and is actively
maintained by Google.

## Getting started

### Enable Stackdriver APIs

Ensure the [Stackdriver Logging][logging_api] and [Stackdriver
Monitoring][monitoring_api] APIs are enabled.

[logging_api]:    https://console.developers.google.com/apis/api/logging.googleapis.com/overview
[monitoring_api]: https://console.developers.google.com/apis/api/monitoring.googleapis.com/overview

#### Quotas

Depending on the size of the cloud foundry deployment and which events the nozzle is forwarding,
it can be quite easy to reach the default Stackdriver quotas:

* [Monitoring Quota](https://cloud.google.com/monitoring/quota-policy)
* [Logging Quota](https://cloud.google.com/logging/quota-policy)

Google quotas can be viewed and managed on the [API Quotas Page](https://console.cloud.google.com/iam-admin/quotas).
An operator can increase the default quota up to a limit; exceeding that, use the contact
links to request even higher quotas.

### Create and configure service accounts

All of the jobs in this release authenticate to Stackdriver Logging and
Monitoring via [Service
Accounts](https://cloud.google.com/compute/docs/access/create-enable-service-accounts-for-instances).
Follow the [GCP documentation](https://cloud.google.com/iam/docs/granting-changing-revoking-access) to create a service account via gcloud with the following roles:

- `roles/logging.logWriter`
- `roles/logging.configWriter`
- `roles/monitoring.metricWriter`

You can either authenticate the job(s) by specifying the service account in the `cloud_properties` for the [resource pool](https://bosh.io/docs/deployment-manifest.html#resource-pools) running the job(s) or by configuring `credentials.application_default_credentials` in the job spec. 

You may also read the [access control
documentation](https://cloud.google.com/logging/docs/access-control) for more
general information about how authentication and authorization work for
Stackdriver.

## General usage

To use any of the jobs in this BOSH release, first upload it to your BOSH
director:

```
bosh2 upload-release https://storage.googleapis.com/bosh-gcp/beta/stackdriver-tools/latest.tgz
```

The [stackdriver-tools.yml][tools-yaml] sample [BOSH 2.0 manifest][bosh20] illustrates how to
use all 3 jobs in this release (nozzle, host logging, and host monitoring). You
can deploy the sample with the following commands:

[tools-yaml]: manifests/stackdriver-tools.yml
[bosh20]: https://bosh.io/docs/manifest-v2.html

```
bosh2 upload-stemcell https://bosh.io/d/stemcells/bosh-google-kvm-ubuntu-trusty-go_agent

bosh2 update-cloud-config -n manifests/cloud-config-gcp.yml \
          -v zone=... \
          -v network=... \
          -v subnetwork=... \
          -v "tags=['stackdriver-nozzle']" \
          -v internal_cidr=... \
          -v internal_gw=... \
          -v "reserved=[10....-10....]"

bosh2 deploy manifests/stackdriver-tools.yml \
            -d stackdriver-nozzle \
            --var=firehose_endpoint=https://.. \
            --var=firehose_username=stackdriver_nozzle \
            --var=firehose_password=... \
            --var=skip_ssl=false \
            --var=gcp_project_id=... \
            --var-file=gcp_service_account_json=path/to/service_account.json \
```

This will create a self-contained deployment that sends Cloud Foundry firehose
data, host logs, and host metrics to Stackdriver. 

Deploying each job individually is described in detail below.

## Deploying the nozzle

Create a new deployment manifest for the nozzle. See the [example
manifest][tools-yaml] for a full deployment and the `jobs.stackdriver-nozzle`
section for the nozzle.

To reduce message loss, operators should run a minimum of two instances. With
two instances, updating stemcells and other destructive BOSH operations will
still leave an instance draining logs.

The [loggregator][loggregator] system will round-robin messages across multiple
instances. If the nozzle can't handle the load, consider scaling to more than
two nozzle instances.

The [spec][spec] describes all the properties an operator should modify.

[spec]: jobs/stackdriver-nozzle/spec
[loggregator]: https://github.com/cloudfoundry/loggregator


### Stackdriver Error Reporting

Stackdriver can automatically detect and report errors from stack traces in logs.
However, this does not automatically work with Loggregator because it sends each
line from app output as a separate log message to the nozzle. To enable this feature 
of Stackdriver, apps will need to manually encode stacktraces on a single line so 
that the stackdriver-nozzle can send them as single messages to Stackdriver.

This is accomplished by replacing newlines in stacktraces with a unique character,
which is set using the `firehose.newline_token` template variable in the nozzle
so that the nozzle can reconstruct the stacktrace on multiple lines.

For example, if `firehose.newline_token` is set to `∴`, a Go app would need to
implement something like the following:

```go
const newlineToken = "∴"

func main() {
    ...
    defer handlePanic()
    ...
}

func handlePanic() {
    	e := recover()
    	if e == nil {
    		return
    	}
    
    	stack := make([]byte, 1<<16)
    	stackSize := runtime.Stack(stack, true)
    	out := string(stack[:stackSize])
    
    	fmt.Fprintf(os.Stderr, "panic: %v", e)
    	fmt.Fprintf(os.Stderr, strings.Replace(out, "\n", newlineToken, -1))
    	os.Exit(1)
}
```

This outputs the stacktrace separately from the panic so that the panic remains in
the logs and the stacktrace is logged by itself. This allows Stackdriver to detect
the stacktrace as an error.

For an example in Java, see [this section of the Loggregator documentation][multi-line-java].

[multi-line-java]: https://github.com/cloudfoundry/loggregator#multi-line-java-message-workaround

## Deploying host logging
The [google-fluentd][google-fluentd] template uses [Fluentd][fluentd] to send
both syslog and template logs (assuming that template jobs are writing logs into
`/var/vcap/sys/log/*/*.log`) to [Stackdriver Logging][logging].

To forward host logs from BOSH VMs to Stackdriver, co-locate the
[google-fluentd] template with an existing job whose host logs should be
forwarded.

[google-fluentd]: jobs/google-fluentd
[stackdriver-agent]: jobs/stackdriver-agent

Include the `stackdriver-tools` release in your existing deployment manifest:

```
releases:
  ...
  - name: stackdriver-tools
    version: latest
  ...
```

Add the [google-fluentd] template to your job:

```
jobs:
  ...
  - name: nats
    templates:
      - name: nats
        release: cf
      - name: metron_agent
        release: cf
      - name: google-fluentd
        release: stackdriver-tools
  ...
```

## Deploying host monitoring
The [stackdriver-agent][stackdriver-agent] template uses the [Stackdriver
Monitoring Agent][monitoring-agent] to collect VM metrics to send to
[Stackdriver Monitoring][monitoring].

To forward host metrics forwarding from BOSH VMs to Stackdriver, co-locate the
[stackdriver-agent] template with an existing job whose host metrics should be
forwarded.

[stackdriver-agent]: jobs/stackdriver-agent

Include the `stackdriver-tools` release in your existing deployment manifest:

```
releases:
  ...
  - name: stackdriver-tools
    version: latest
  ...
```

Add the [stackdriver-agent] template to your job:

```
jobs:
  ...
  - name: nats
    templates:
      - name: nats
        release: cf
      - name: metron_agent
        release: cf
      - name: stackdriver-agent
        release: stackdriver-tools
  ...
```

## Deploying as a BOSH addon
Specify the jobs as addons in your [runtime config](https://bosh.io/docs/runtime-config.html) to deploy Stackdriver Monitoring and Logging agents on all instances in your deployment. Do not specify the jobs as part of your deployment manifest if you are using the runtime config.

```
# runtime.yml
---
releases:
  - name: stackdriver-tools
    version: latest

addons:
- name: stackdriver-tools
  jobs:
  - name: google-fluentd
    release: stackdriver-tools
  - name: stackdriver-agent
    release: stackdriver-tools
```

To deploy the runtime config:

```
bosh update runtime-config runtime.yml
bosh deploy
```

## Development

### Updating google-fluentd

`google-fluentd` is versioned by the [Gemfile in src/google-fluentd][gemfile]. To update [fluentd][fluentd]:

1. Update the version specifier in the Gemfile (if necessary)
1. Update Gemfile.lock: `bundle update`
1. Create a vendor cache from the Gemfile.lock: `bundle package`
1. Tar and compress the vendor folder: `tar zvc vendor >
   google-fluentd-vendor-VERSION-NUMBER.tgz`
1. Update the vendor version in the `google-fluentd` package
   [packaging][fluentd-packaging] and [spec][fluentd-spec]
1. Add vendored cache to the BOSH blobstore: `bosh add blob
   google-fluentd-vendor-VERSION-NUMBER.tgz google-fluentd-vendor`
1. [Create a dev release][dev-release] and deploy it to verify that all of the
   above worked
1. Update the BOSH blobstore: `bosh upload blobs`
1. Commit your changes

[gemfile]: https://github.com/cloudfoundry-community/stackdriver-tools/blob/master/src/google-fluentd/Gemfile
[fluentd]: https://github.com/fluent/fluentd
[fluentd-packaging]: https://github.com/cloudfoundry-community/stackdriver-tools/blob/master/packages/google-fluentd/packaging
[fluentd-spec]: https://github.com/cloudfoundry-community/stackdriver-tools/blob/master/packages/google-fluentd/spec
[dev-release]: https://bosh.io/docs/create-release.html#dev-release

### bosh-lite

Both the nozzle and the fluentd jobs can run on [bosh-lite][bosh-lite]. To generate a working manifest, start from
the [bosh-lite-example-manifest][bosh-lite-example-manifest]. Note the `application_default_credentials`
property, which should be filled in with the contents of a [Google service account key][google-service-account-key].

[bosh-lite]: https://github.com/cloudfoundry/bosh-lite
[bosh-lite-example-manifest]: manifests/stackdriver-tools-bosh-lite.yml
[google-service-account-key]: https://cloud.google.com/compute/docs/access/create-enable-service-accounts-for-instances

## Contributing
For detailes on how to contribute to this project - including filing bug reports
and contributing code changes - please see [CONTRIBUTING.md].

## Copyright
Copyright (c) 2016 Ferran Rodenas. See
[LICENSE](https://github.com.evandbrown/stackdriver-tools/blob/master/LICENSE)
for details.

[CONTRIBUTING.md]: CONTRIBUTING.md

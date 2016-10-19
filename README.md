# Google Cloud Platform Tools BOSH Release

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

This is currently a beta release. It should be used in production environments
with an abundance of caution, and only after being vetted in dev environment.

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
You must create a service account with the following roles:

- `roles/logging.logWriter` to stream logs to Stackdriver Logging
- `roles/logging.configWriter` to setup CloudFoundry specific metrics on
  Stackdriver Monitoring

The BOSH resource pool you deploy the job(s) to must use that service account
by specifying it in `cloud_properties`. The [BOSH Google CPI
documentation](https://cloud.google.com/compute/docs/access/create-enable-service-accounts-for-instances)
describes how to set the `service_account` for a resource pool.

You may also read the [access control
documentation](https://cloud.google.com/logging/docs/access-control) for more
general information about how authentication and authorization work for
Stackdriver.

## General usage

To use any of the jobs in this BOSH release, first upload it to your BOSH
director:

```
beta_release=https://storage.googleapis.com/bosh-gcp/beta/bosh-gcp-tools-$(curl -s https://storage.googleapis.com/bosh-gcp/beta/current-version).tgz
bosh target BOSH_HOST
bosh upload ${beta_release}
```

The [gcp-tools.yml][tools-yaml] sample deployment manifest illustrates how to
use all 3 jobs in this release (nozzle, host logging, and host monitoring). You
can deploy the sample with:

[tools-yaml]: manifests/gcp-tools.yml


```
bosh deployment manifests/gcp-tools.yml 
bosh -n deploy
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

## Deploying host logging
The [google-fluentd][google-fluentd] template uses [Fluentd][fluentd] to send
both syslog and template logs (assuming that template jobs are writing logs into
`/var/vcap/sys/log/*/*.log`) to [Stackdriver Logging][logging].

To forward host logs from BOSH VMs to Stackdriver, co-locate the
[google-fluentd] template with an existing job whose host logs should be
forwarded.

[google-fluentd]: jobs/google-fluentd
[stackdriver-agent]: jobs/stackdriver-agent

Include the `bosh-gcp-tools` release in your existing deployment manifest:

```
releases:
  ...
  - name: bosh-gcp-tools
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
        release: bosh-gcp-tools
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

Include the `bosh-gcp-tools` release in your existing deployment manifest:

```
releases:
  ...
  - name: bosh-gcp-tools
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
        release: bosh-gcp-tools
  ...
```

## Deploying as a BOSH addon
Specify the jobs as addons in your [runtime config](https://bosh.io/docs/runtime-config.html) to deploy Stackdriver Monitoring and Logging agents on all instances in your deployment. Do not specify the jobs as part of your deployment manifest if you are using the runtime config.

```
# runtime.yml
---
releases:
  - name: bosh-gcp-tools
    version: latest

addons:
- name: gcp-tools
  jobs:
  - name: google-fluentd
    release: bosh-gcp-tools
  - name: stackdriver-agent
    release: bosh-gcp-tools
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

[gemfile]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/src/google-fluentd/Gemfile
[fluentd]: https://github.com/fluent/fluentd
[fluentd-packaging]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/packages/google-fluentd/packaging
[fluentd-spec]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/packages/google-fluentd/spec
[dev-release]: https://bosh.io/docs/create-release.html#dev-release

## Contributing
For detailes on how to contribute to this project - including filing bug reports
and contributing code changes - please see [CONTRIBUTING.md].

## Copyright
Copyright (c) 2016 Ferran Rodenas. See
[LICENSE](https://github.com.evandbrown/gcp-tools-release/blob/master/LICENSE)
for details.

[CONTRIBUTING.md]: CONTRIBUTING.md

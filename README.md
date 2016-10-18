# Google Cloud Platform Tools BOSH Release

This is a [BOSH](http://bosh.io/) release for [Google Cloud Platform](https://cloud.google.com/) Tools. This release
contains the following templates:

* [Fluentd][fluentd] for forwarding syslog and template logs to [Stackdriver Logging][logging]
* The [Stackdriver Monitoring Agent][monitoring-agent] for sending VM health metrics to [Stackdriver Monitoring][monitoring]
* A [stackdriver-nozzle][nozzle] for forwarding [Cloud Foundry Firehose][firehose] data to Stackdriver

[monitoring]: https://cloud.google.com/monitoring/
[fluentd]: http://www.fluentd.org/
[monitoring-agent]: https://cloud.google.com/monitoring/agent/
[logging]: https://cloud.google.com/logging/
[firehose]: https://docs.cloudfoundry.org/loggregator/architecture.html#firehose
[nozzle]: src/stackdriver-nozzle

## Disclaimer

This project is currently in **BETA**. Use in production at your own risk.

## Access Control

The following roles are required for the service account on each deployed instance:

 - `roles/logging.logWriter` to stream logs to Stackdriver Logging
 - `roles/logging.configWriter` to setup CloudFoundry specific metrics on Stackdriver Monitoring

See the [access control documentation](https://cloud.google.com/logging/docs/access-control) for more information.

## Enabled Services

To use Stackdriver Monitoring ensure the [Stackdriver Monitoring API][stackdriver_api] is enabled.

[stackdriver_api]: https://console.developers.google.com/apis/api/monitoring.googleapis.com/overview

## Usage

To use this BOSH release, first upload it to your BOSH:

```
bosh target BOSH_HOST
bosh upload https://storage.googleapis.com/bosh-releases/gcp-tools-1.tgz
```

See [manifests/gcp-tools.yml][tools-yaml] for a sample deployment manifest that can be used as a starting point.

```
bosh deployment manifests/gcp-tools.yml 
bosh -n deploy
```

This will create a self-contained deployment that collects VM data from itself and CF data from the Firehose into
Stackdriver.

### Stackdriver Logging

Once deployed:
* the `google-fluentd` will act as a Syslog endpoint and will forward logs to [Stackdriver Logging][logging]

If you want to send all your Cloud Foundry component's logs to [Stackdriver Logging][logging], configure your Cloud
Foundry manifest adding (or updating):

```
properties:
  ...
  syslog_daemon_config:
    address: <google-fluentd job instance IP address>
    port: 514
    transport: udp
```

[tools-yaml]: manifests/gcp-tools.yml

### Stackdriver Monitoring

Add the `gcp-tools` release to the `release` section of your existing deployment manifest:

```
release:
   ...
  - name: gcp-tools
    version: "1"
```

Collocate the `stackdriver-agent` job template in all job instances:

```
jobs:
  - name: nats
    templates:
      - name: nats
        release: cf
      - name: metron_agent
        release: cf
      - name: stackdriver-agent
        release: gcp-tools
```

Once deployed, the `stackdriver-agent` on every instance will send host metrics to [Stackdriver Monitoring][monitoring].

### stackdriver-nozzle

Create a new deployment manifest for the nozzle. See the [example manifest][tools-yaml] 
for a full deployment and the `jobs.stackdriver-nozzle` section for the nozzle.

To reduce message loss, operators should run a minimum of two instances. With two instances,
updating stemcells and other destructive BOSH operations will still leave an instance
draining logs.

The [loggregator][loggregator] system will round-robin messages across multiple instances. If the
nozzle can't handle the load, consider scaling to more than two nozzle instances.

The [spec][spec] describes all the properties an operator should modify.

[spec]: jobs/stackdriver-nozzle/spec
[loggregator]: https://github.com/cloudfoundry/loggregator

## Development

### Updating google-fluentd

`google-fluentd` is versioned by the [Gemfile in src/google-fluentd][gemfile]. To update [fluentd][fluentd]:

1. Update the version specifier in the Gemfile (if necessary)
1. Update Gemfile.lock: `bundle update`
1. Create a vendor cache from the Gemfile.lock: `bundle package`
1. Tar and compress the vendor folder: `tar zvc vendor > google-fluentd-vendor-VERSION-NUMBER.tgz`
1. Update the vendor version in the `google-fluentd` package [packaging][fluentd-packaging] and [spec][fluentd-spec]
1. Add vendored cache to the BOSH blobstore: `bosh add blob google-fluentd-vendor-VERSION-NUMBER.tgz google-fluentd-vendor`
1. [Create a dev release][dev-release] and deploy it to verify that all of the above worked
1. Update the BOSH blobstore: `bosh upload blobs`
1. Commit your changes

[gemfile]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/src/google-fluentd/Gemfile
[fluentd]: https://github.com/fluent/fluentd
[fluentd-packaging]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/packages/google-fluentd/packaging
[fluentd-spec]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/packages/google-fluentd/spec
[dev-release]: https://bosh.io/docs/create-release.html#dev-release

### Contributing

In the spirit of [free software][free-sw], **everyone** is encouraged to help improve this project.

[free-sw]: http://www.fsf.org/licensing/essays/free-sw.html

Here are some ways *you* can contribute:

* by using alpha, beta, and pre-release versions
* by reporting bugs
* by suggesting new features
* by writing or editing documentation
* by writing tests
* by writing code (**no patch is too small**: fix typos, add comments, clean up inconsistent whitespace)
* by reviewing patches

### Submitting an Issue

We use the [GitHub issue tracker][issues] to track bugs and features. Before submitting a bug report or feature request,
check to make sure it hasn't already been submitted. You can indicate support for an existing issue by voting it up.
When submitting a bug report, please include a [Gist](http://gist.github.com/) that includes a stack trace and any
details that may be necessary to reproduce the bug,. Ideally, a bug report should include a pull request with failing
specs.

[issues]: https://github.com/cloudfoundry-community/gcp-tools-release/issues

### Submitting a Pull Request

1. Fork the project.
2. Create a topic branch.
3. Implement your feature or bug fix.
4. Commit and push your changes.
5. Submit a pull request.

## Copyright

Copyright (c) 2016 Ferran Rodenas. See [LICENSE](https://github.com.evandbrown/gcp-tools-release/blob/master/LICENSE) for details.

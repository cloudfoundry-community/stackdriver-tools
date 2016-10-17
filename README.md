# Google Cloud Platform Tools BOSH Release

This is a [BOSH](http://bosh.io/) release for [Google Cloud Platform](https://cloud.google.com/) Tools:

* A job acting as a Syslog endpoint to send platform logs to [Stackdriver Logging](https://cloud.google.com/logging/)
* A job that forwards [Cloud Foundry Firehose](https://docs.cloudfoundry.org/loggregator/architecture.html#firehose) event data (including application logs) to [Stackdriver Logging](https://cloud.google.com/logging/)
* A job (to be collocated with other jobs) to send host metrics to [Stackdriver Monitoring](https://cloud.google.com/monitoring/)

## Disclaimer

** This BOSH release can only be deployed to [Google Cloud Platform](https://cloud.google.com/)**

This is NOT presently a production ready BOSH release. This is just a Proof of Concept. It is suitable for experimentation and may not become supported in the future.

## Access Control

The follow roles are required for the service account on each deployed instance:

 - `roles/logging.logWriter` to stream logs to Stackdriver Logging
 - `roles/logging.configWriter` to setup CloudFoundry specific metrics on Stackdriver Monitoring

See the [access control documentation](https://cloud.google.com/logging/docs/access-control) for more information.

## Enabled Services

To use Stackdriver Monitoring ensure the [Stackdriver Monitoring API](https://console.developers.google.com/apis/api/monitoring.googleapis.com/overview) is enabled.

## Usage

### Upload the BOSH release

To use this BOSH release, first upload it to your BOSH:

```
bosh target BOSH_HOST
bosh upload https://storage.googleapis.com/bosh-releases/gcp-tools-1.tgz
```

### Deploying Stackdriver Logging

Create a deployment file (use the [gcp-tools.yml](https://github.com.evandbrown/gcp-tools-release/blob/master/manifests/gcp-tools.yml) example manifest file as a starting point).

Using the previous created deployment manifest, now deploy it:

```
bosh deployment path/to/deployment.yml
bosh -n deploy
```

Once deployed:
* the `google-fluentd` will act as a Syslog endpoint and will forward logs to [Stackdriver Logging](https://cloud.google.com/logging/)

If you want to send all your Cloud Foundry component's logs to [Stackdriver Logging](https://cloud.google.com/logging/), configure your Cloud Foundry manifest adding (or updating):

```
properties:
  ...
  syslog_daemon_config:
    address: <google-fluentd job instance IP address>
    port: 514
    transport: udp
```

### Deploying Stackdriver Monitoring

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

Once deployed, the `stackdriver-agent` on every instance will send host metrics to 
[Stackdriver Monitoring](https://cloud.google.com/monitoring/).

## Development

### Updating google-fluentd

`google-fluentd` is versioned by the [Gemfile in src/google-fluentd][gemfile]. To update [fluentd][fluentd]:

1. Update the version specifier in the Gemfile (if necessary)
1. Update Gemfile.lock: `bundle update`
1. Create a vendor cache from the Gemfile.lock: `bundle package`
1. Tar and compress the vendor folder: `tar zvc vendor > google-fluentd-vendor-VERSION-NUMBER.tgz`
1. Update the vendor version in the `google-fluentd` package [packaging][packaging] and [spec][spec]
1. Add vendored cache to the BOSH blobstore: `bosh add blob google-fluentd-vendor-VERSION-NUMBER.tgz google-fluentd-vendor`
1. [Create a dev release][dev-release] and deploy it to verify that all of the above worked
1. Update the BOSH blobstore: `bosh upload blobs`
1. Commit your changes

[gemfile]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/src/google-fluentd/Gemfile
[fluentd]: https://github.com/fluent/fluentd
[packaging]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/packages/google-fluentd/packaging
[spec]: https://github.com/cloudfoundry-community/gcp-tools-release/blob/master/packages/google-fluentd/spec
[dev-release]: https://bosh.io/docs/create-release.html#dev-release

### Contributing

In the spirit of [free software](http://www.fsf.org/licensing/essays/free-sw.html), **everyone** is encouraged to help improve this project.

Here are some ways *you* can contribute:

* by using alpha, beta, and prerelease versions
* by reporting bugs
* by suggesting new features
* by writing or editing documentation
* by writing specifications
* by writing code (**no patch is too small**: fix typos, add comments, clean up inconsistent whitespace)
* by refactoring code
* by closing [issues](https://github.com.evandbrown/gcp-tools-release/issues)
* by reviewing patches

### Submitting an Issue

We use the [GitHub issue tracker](https://github.com.evandbrown/gcp-tools-release/issues) to track bugs and features. Before submitting a bug report or feature request, check to make sure it hasn't already been submitted. You can indicate support for an existing issue by voting it up. When submitting a bug report, please include a [Gist](http://gist.github.com/) that includes a stack trace and any details that may be necessary to reproduce the bug,. Ideally, a bug report should include a pull request with failing specs.

### Submitting a Pull Request

1. Fork the project.
2. Create a topic branch.
3. Implement your feature or bug fix.
4. Commit and push your changes.
5. Submit a pull request.

## Copyright

Copyright (c) 2016 Ferran Rodenas. See [LICENSE](https://github.com.evandbrown/gcp-tools-release/blob/master/LICENSE) for details.

## stackdriver-nozzle

A service that connects to the [Cloud Foundry firehose][cf-firehose] and sends
logs and metrics to [Google Stackdriver][goog-sd].

[cf-firehose]: https://docs.cloudfoundry.org/loggregator/architecture.html
[goog-sd]: https://cloud.google.com/stackdriver/

### Installation

```sh
go get github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle
```

### Configuration

`stackdriver-nozzle` is configured through the following environment variables:

#### Firehose

- `FIREHOSE_ENDPOINT` - the CF API endpoint; e.g., `https://api.bosh-lite.com'
- `FIREHOSE_EVENTS_TO_STACKDRIVER_LOGGING` - comma-separated list of events to pass to Stackdriver Logging;
  valid events are `LogMessage`,  `Error`, `HttpStartStop`, `ValueMetric` (BETA), `CounterEvent` (BETA),
  `ContainerMetric` (BETA)
- `FIREHOSE_EVENTS_TO_STACKDRIVER_MONITORING` - comma-separated list of events to pass to Stackdriver Monitoring;
  valid events are  `ValueMetric`, `CounterEvent`, and `ContainerMetric`
- `FIREHOSE_USERNAME` - CF username; defaults to `admin`
  - requires `doppler.firehose` and `cloud_controller.admin_read_only` permissions
- `FIREHOSE_PASSWORD` - CF password; defaults to `password`
- `FIREHOSE_SKIP_SSL` - whether to ignore SSL (please don't); defaults to
  `false`

#### Stackdriver

- `GCP_PROJECT_ID` - the GCP project ID; will be automatically configured from
  the environment using [metadata][metadata] if left empty

[metadata]: https://cloud.google.com/compute/docs/storing-retrieving-metadata

#### Nozzle

- `HEARTBEAT_RATE` - how often `stackdriver-nozzle` reports stats to stdout;
  defaults to 30 seconds
- `LOGGING_BATCH_COUNT` - how many logs to batch into a single report to
  Stackdriver; defaults to 10
- `LOGGING_BATCH_DURATION` - maximum time to batch logs to Stackdriver; defaults to 1
  second
- `METRICS_BUFFER_DURATION` - flush interval (in seconds) of the internal metric
  buffer; defaults to 30
- `METRICS_BATCH_SIZE` - batch size for metric time series being sent to
  Stackdriver; defaults to 200
- `RESOLVE_APP_METADATA` - whether to hydrate app UUIDs into org name, org
  UUID, space name, space UUID, and app name; defaults to `true`
- `SUBSCRIPTION_ID` - what subscription ID to use for connecting to the
  firehose; defaults to `stackdriver-nozzle`

### Usage

```sh
go run main.go
```

### Development

Run `make newb` to install required development dependencies.

A [.envrc.template][envrc-template] template is provided for a quick setup. We
suggest copying it to `.envrc` and using [direnv][direnv] to automatically set
the environment variables when you're in the `stackdriver-nozzle` directory.

[envrc-template]: https://github.com/cloudfoundry-community/stackdriver-tools/blob/master/src/stackdriver-nozzle/.envrc.template
[direnv]: http://direnv.net/

When running on GCP, `stackdriver-nozzle` will [automatically configure][dts]
the required credentials and project ID, but they will need to be provided
manually through environment variables (see above) when running locally. You
can get the credentials JSON file by following [Google's instructions
here][goog-creds] from the [credentials console][cred-console].

[dts]: https://godoc.org/golang.org/x/oauth2/google#DefaultTokenSource
[goog-creds]: https://developers.google.com/identity/protocols/application-default-credentials
[cred-console]: https://console.developers.google.com/project/_/apis/credentials

[Ginkgo][ginkgo] is used for testing.

[ginkgo]: https://github.com/onsi/ginkgo

```sh
ginkgo -r
```

### Dependencies

This project uses [govendor](https://github.com/kardianos/govendor) for
dependency management.

```
govendor fetch +missing
```

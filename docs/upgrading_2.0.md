# Upgrade Procedure for 2.0

This release of `stackdriver-tools` introduces significant breaking changes for `stackdriver-nozzle` users.
These breaking changes provide improved metrics functionality but require the operator to reset 
aspects of their project. This will result in irreversible historical data loss and the need to 
recreate dashboards.

## Stackdriver Monitoring Changes
All exported metrics have changed in this release.

Metric changes:
- Metrics derived from the firehose now have the metric origin prefixed to the metric name, e.g. "gorouter.total_requests".
- Metrics derived from the firehose are now exported under a configurable path prefix, which defaults to "firehose".
- CounterEvent metrics are now represented as COUNTERs rather than GAUGEs in Stackdriver, so per-second rates can be computed.
- The nozzle exports its own custom metrics under the "stackdriver-nozzle" prefix.

Label changes:
- Application, space and organization UUIDs are no longer exported.
- Resolved application, space and organization names are concatenated into a single "applicationPath" label.
- The application instance index is now exported.
- Event envelope tags are now exported as a string of comma-separated key=value pairs.
- Origin and EventType labels are only attached to logs, not metrics.
- There is a new "foundation" label that is configurable per nozzle instance, to distinguish between multiple foundations in a GCP / Stackdriver project.

## Clear Metric Descriptors
Follow [this guide](../src/stackdriver-nozzle/docs/clear-metrics-descriptors.md) to reset metric descriptors
for your Stackdriver Monitoring project. This needed is to stay under the default [custom metric descriptor](https://cloud.google.com/monitoring/custom-metrics/) quota.
Ensure the `stackdriver-nozzle` is not running during this process.

An alternative to this procedure is to provision a new Google Cloud Project and configure the updated `stackdriver-nozzle` release to use it.

## Singleton Deployment of `stackdriver-nozzle`
The `stackdriver-nozzle` keeps track of counter metrics to send to Stackdriver Monitoring at the instance level by default. 
If multiple copies of the `stackdriver-nozzle` are reporting the same metrics for a Cloud Foundry deployment they
will report incorrect data. This issue is mitigated by running a single instance of the nozzle. 
An upstream feature in Loggregator is being [tracked](https://www.pivotaltracker.com/n/projects/993188/stories/154821450) to address this.

Operations Manager/Tile users are forced to run the `stackdriver-nozzle` job as a single instance. 

BOSH release users must ensure they are running a single instance and should review the [example manifest](../manifests/stackdriver-tools.yml).

Users are encouraged to vertically scale the `stackdriver-nozzle` VM in case of resource saturation. A number of performance optimizations in this release make sure 
that a single instance of the nozzle is able to provide enough throughput for a medium sized PCF installation.

If you find the need to scale the `stackdriver-nozzle` job horizontally you must use the BOSH release. There are two possible ways forward:

- If the nozzle is heavily used to proxy log events, you can run two separate nozzle bosh jobs: 
  one for log messages (which can be scaled horizontally), and the other for metrics (which will need to have a single instance). 
  Nozzle jobs can be configured to receive only a subset of events via `firehose.events_to_stackdriver_logging` and 
  `firehose.events_to_stackdriver_monitoring` BOSH manifest properties.
- You can disable cumulative counter tracking completely by setting `nozzle.enable_cumulative_counters` to `false`. 
  This will revert to the old behavior of exporting each counter as a pair of gauges (with `.delta` and `.total` suffixes). 
  With cumulative counters disabled, a single nozzle job can be configured to run with multiple instances


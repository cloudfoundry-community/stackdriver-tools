# Upgrade Procedure for 2.0

This release of `stackdriver-tools` introduces significant breaking changes for `stackdriver-nozzle` users.
These breaking changes provide improved metrics functionality but require the operator to reset 
aspects of their project. This will result in irreversible historical data loss and the need to 
recreate dashboards.

This release of the `stackdriver-nozzle` changes the name, type, and labels of all reported metrics.

## Clear Metric Descriptors
Follow [this guide](../src/stackdriver-nozzle/docs/clear-metrics-descriptors.md) to reset metric descriptors
for your Stackdriver Monitoring project. This needed is to stay under the default [custom metric descriptor](https://cloud.google.com/monitoring/custom-metrics/) quota.
Ensure the `stackdriver-nozzle` is not running during this process.

## Singleton Deployment of `stackdriver-nozzle`
The `stackdriver-nozzle` keeps track of counter metrics to send to Stackdriver Monitoring at the instance level by default. 
If multiple copies of the `stackdriver-nozzle` are reporting the same metrics for a Cloud Foundry deployment they
will report incorrect data. 

This issue is mitigated by running a single instance of the nozzle. 
An upstream feature in Loggregator is being [tracked](https://www.pivotaltracker.com/n/projects/993188/stories/154821450) to fix this.
It is highly recommended that users evaluate running a single instance. This release significantly improves
performance of the `stackdriver-nozzle` and will reduce the needed footprint compared to earlier releases.

If you find that a single instance is not performing as needed then the manifest setting `nozzle.enable_cumulative_counters` 
can be set to false. Note this option is not avaliable to tile users and will be removed once the tracked
issue is resolved. It is also possible to manually shard to multiple nozzle instances by only subscribing
to specific events (see: `firehose.events_to_stackdriver_monitoring`).
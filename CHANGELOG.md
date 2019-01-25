# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)
and this project adheres to [Semantic Versioning](http://semver.org/spec/v2.0.0.html).

## [2.1.0] - 2019-01-17

### Changed

 - Stackdriver Nozzle PCF Tile requires PCF version >= 2.3.0
 - Stemcell updated from Trusty to Xenial 170.13
 - Stackdriver Nozzle and Spinner updated to use Go 1.11
 - Stackdriver Nozzle and Spinner use dep 
 - Stackdriver Nozzle and Spinner Go dependencies have been updated
 - Various lint fixes were applied across source code and packaged scripts
 - Stackdriver Nozzle utilizes Loggregator’s Reverse Log Proxy API, instead of the Firehose API, for better performance
 - Add foundation lable to Stackdriver Spinner result log to help distinguish betwene multiple spinners
 - Add a syslog endpoint to the fluentd BOSH job for forwarding any syslog through the job to Stackdriver Logging
 - Remove the duplication of the log message field in forwarded CF application logs


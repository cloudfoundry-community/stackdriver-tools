package main

/*
probably:

by configuration:
- /varz like output
- force a run via HTTP

for sure:
- http endpoint
- stackdriver monitoring, logging client
- logging client
- probe:
  - emit logs
  - pool stackdriver w/ timeout
  - emit metric results

packages:
- stackdriver: clients
  - MonitoringClient
  - LoggingProbe
       Find(needle) count
- cloudfoundry:
  - Logger:
		Emit(needle, count)
  - stdoutLogger
- session:
  - NewSession
*/

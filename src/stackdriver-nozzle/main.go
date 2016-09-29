package main

import (
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/firehose"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/dev"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	debug = kingpin.Flag("debug", "send events to stdout").
		Default("false").
		OverrideDefaultFromEnvar("DEBUG").
		Bool()
	apiEndpoint = kingpin.Flag("api-endpoint",
		"CF API endpoint (use https://api.bosh-lite.com for BOSH Lite)").
		OverrideDefaultFromEnvar("API_ENDPOINT").
		Required().
		String()
	username = kingpin.Flag("username", "username").
			Default("admin").
			OverrideDefaultFromEnvar("FIREHOSE_USERNAME").
			String()
	password = kingpin.Flag("password", "password").
			Default("admin").
			OverrideDefaultFromEnvar("FIREHOSE_PASSWORD").
			String()
	skipSSLValidation = kingpin.Flag("skip-ssl-validation", "please don't").
				Default("false").
				OverrideDefaultFromEnvar("SKIP_SSL_VALIDATION").
				Bool()
	projectID = kingpin.Flag("project-id", "gcp project id").
			OverrideDefaultFromEnvar("PROJECT_ID").
			String() //maybe we can get this from gcp env...? research
)

func main() {
	//todo: pull in logging library...
	kingpin.Parse()

	client := firehose.NewClient(*apiEndpoint, *username, *password, *skipSSLValidation)

	if *debug {
		println("Sending firehose to standard out")
		err := client.StartListening(&dev.StdOut{})
		if err != nil {
			panic(err)
		}
	} else {
		println("Sending firehose to Stackdriver")
		sdClient := stackdriver.NewClient(*projectID)
		n := nozzle.Nozzle{StackdriverClient: sdClient}

		client.StartListening(&n)
	}
}

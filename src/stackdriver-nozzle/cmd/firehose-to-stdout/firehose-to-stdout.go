/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/cloudfoundry-community/go-cfclient"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
)

func main() {
	apiEndpoint := os.Getenv("FIREHOSE_ENDPOINT")
	username := os.Getenv("FIREHOSE_USERNAME")
	password := os.Getenv("FIREHOSE_PASSWORD")
	_, skipSSLValidation := os.LookupEnv("FIREHOSE_SKIP_SSL")

	cfConfig := &cfclient.Config{
		ApiAddress:        apiEndpoint,
		Username:          username,
		Password:          password,
		SkipSslValidation: skipSSLValidation}

	cfClient, err := cfclient.NewClient(cfConfig)
	if err != nil {
		panic(err)
	}

	client := cloudfoundry.NewFirehose(cfConfig, cfClient, "")

	firehose, errorhose := client.Connect()
	if firehose == nil {
		panic(errors.New("firehose was nil"))
	} else if errorhose == nil {
		panic(errors.New("errorhose was nil"))
	}

	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, os.Interrupt)

	errorLog := log.New(os.Stderr, "", 0)

	for {
		select {
		case envelope := <-firehose:
			if envelope == nil {
				errorLog.Println("received nil envelope")
			} else {
				println(envelope.String())
			}
		case err := <-errorhose:
			if err == nil {
				errorLog.Println("received nil envelope")
			} else {
				errorLog.Println(err)
			}
		case <-exitSignal:
			os.Exit(0)
		}
	}
}

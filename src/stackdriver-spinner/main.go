/*
 * Copyright 2019 Google Inc.
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
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/session"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/stackdriver"
)

func main() {

	count, err := strconv.Atoi(os.Getenv("SPINNER_COUNT"))
	if err != nil {
		log.Fatal(err)
	}

	wait, err := strconv.Atoi(os.Getenv("SPINNER_WAIT"))
	if err != nil {
		log.Fatal(err)
	}

	gcpProj := os.Getenv("GCP_PROJECT")
	if len(gcpProj) == 0 {
		log.Fatal("A GCP project must be specified.")
	}

	foundation := os.Getenv("FOUNDATION")
	if len(foundation) == 0 {
		log.Fatal("A foundation must be specified.")
	}

	go startSpinner(gcpProj, foundation, count, wait)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		if _, err := fmt.Fprintf(res, "Johny 5 alive!"); err != nil {
			log.Fatal(err)
		}
	})
	fmt.Println("listening...")

	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func startSpinner(proj, foundation string, count, wait int) {
	burstInterval := time.Duration(wait) * time.Second

	emitter := cloudfoundry.NewEmitter(os.Stdout, count, 10*time.Millisecond)
	probe, err := stackdriver.NewLoggingProbe(proj)
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(emitter, probe)
	for {
		result, err := s.Run(burstInterval)
		if err != nil {
			log.Println(err)
			continue
		}
		logger, err := stackdriver.NewLogger(proj, foundation)
		if err != nil {
			log.Fatal(err)
		}

		msg := stackdriver.Message{
			GUID:             result.GUID,
			NumberSent:       count,
			NumberFound:      result.Found,
			BurstIntervalSec: wait,
			LossPercentage:   result.Loss,
		}

		logger.Publish(msg)

	}
}

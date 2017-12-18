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

	go startSpinner(gcpProj, count, wait)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "Johny 5 alive!")
	})
	fmt.Println("listening...")

	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func startSpinner(proj string, count, wait int) {
	burstInterval := time.Duration(wait) * time.Second

	emitter := cloudfoundry.NewEmitter(os.Stdout)
	probe, err := stackdriver.NewLoggingProbe(proj)
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(emitter, probe)
	for {
		result, err := s.Run(count, burstInterval, 10*time.Millisecond)
		if err != nil {
			log.Println(err)
			continue
		}
		logger, err := stackdriver.NewLogger(proj)

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

package main

import (
	"os"

	"log"

	"time"

	"strconv"

	"fmt"
	"net/http"

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

	go startSpinner(count, wait)

	http.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(res, "Hello")
	})
	fmt.Println("listening...")

	err = http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal(err)
	}
}

func startSpinner(count, wait int) {
	waitTime := time.Duration(wait) * time.Second

	emitter := cloudfoundry.NewEmitter(os.Stdout)
	probe, err := stackdriver.NewLoggingProbe("cf-cloudops-sandbox")
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(emitter, probe)
	for {
		_, err = s.Run(count, waitTime)
		if err != nil {
			log.Println(err)
		}

	}
}

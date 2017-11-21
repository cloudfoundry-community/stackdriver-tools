package main

import (
	"os"

	"time"

	"log"

	"flag"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/session"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/stackdriver"
)

func main() {

	var count = flag.Int("count", 10, "Number of logs to emit for each generation")
	var waitTime = flag.Duration("waitTime", time.Second, "Interval between log probes")
	flag.Parse()

	emitter := cloudfoundry.NewEmitter(os.Stdout)
	probe, err := stackdriver.NewLoggingProbe("cf-cloudops-sandbox")
	if err != nil {
		log.Fatal(err)
	}
	s := session.NewSession(emitter, probe)
	for {
		_, err = s.Run(*count, *waitTime)
		if err != nil {
			log.Println(err)
		}

	}
}

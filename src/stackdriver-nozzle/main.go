package main

import "github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/app"

func main() {
	a := app.NewApp()

	producer := a.Producer()
	consumer := a.Consumer()

	errs, fhErrs := consumer.Start(producer)
	defer consumer.Stop()

	go func() {
		for err := range errs {
			a.Logger.Error("nozzle", err)
		}
	}()

	fatalErr := <-fhErrs
	if fatalErr != nil {
		a.Logger.Fatal("firehose", fatalErr)
	}
}

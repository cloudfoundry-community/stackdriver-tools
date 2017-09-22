package app

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
	"github.com/cloudfoundry/lager"
)

func Run(ctx context.Context, a *App) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if a.c.DebugNozzle {
		defer handleFatalError(a, cancel)

		go func() {
			a.logger.Info("pprof", lager.Data{
				"http.ListenAndServe": http.ListenAndServe("localhost:6060", nil),
			})
		}()
	}

	producer := a.newProducer()
	consumer, err := a.newConsumer(ctx)
	if err != nil {
		a.logger.Fatal("construction", err)
	}

	errs, fhErrs := consumer.Start(producer)
	defer func() {
		if err := consumer.Stop(); err != nil {
			a.logger.Error("nozzle.stop", err)
		}
	}()

	go func() {
		for err := range errs {
			a.logger.Error("nozzle", err)
		}
	}()

	if fatalErr := <-fhErrs; fatalErr != nil {
		cancel()
		t := time.NewTimer(5 * time.Second)
		for {
			select {
			case <-time.Tick(100 * time.Millisecond):
				if a.bufferEmpty() {
					a.logger.Fatal("firehose", fatalErr, lager.Data{"cleanup": "The metrics buffer was successfully flushed before shutdown"})
				}
			case <-t.C:
				a.logger.Fatal("firehose", fatalErr, lager.Data{"cleanup": "The metrics buffer could not be flushed before shutdown"})
			}
		}
	}
}

func handleFatalError(a *App, cancel context.CancelFunc) {
	if e := recover(); e != nil {
		// Cancel the context
		cancel()

		stack := make([]byte, 1<<16)
		stackSize := runtime.Stack(stack, true)
		stackTrace := string(stack[:stackSize])

		payload := map[string]interface{}{
			"serviceContext": map[string]interface{}{
				"service": version.Name,
				"version": version.Release(),
			},
			"message": fmt.Sprintf("%v\n%v", e, stackTrace),
		}

		log := &messages.Log{
			Payload:  payload,
			Labels:   map[string]string{},
			Severity: logging.Error,
		}

		// Purposefully get a new log adapter here since there
		// were issues re-using the one that the nozzle uses.
		logAdapter := a.newLogAdapter()
		logAdapter.PostLog(log)
		logAdapter.Flush()

		// Re-throw the error, we want to ensure it's logged directly to
		// stackdriver but we are not in a recoverable state.
		panic(e)
	}
}

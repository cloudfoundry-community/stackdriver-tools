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

package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"cloud.google.com/go/logging"
	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/version"
)

func Run(ctx context.Context, a *App) {
	ctx, cancel := context.WithCancel(ctx)

	if a.c.DebugNozzle {
		defer handleFatalError(a, cancel)

		go func() {
			a.logger.Info("debug", lager.Data{
				"http.ListenAndServe": http.ListenAndServe("0.0.0.0:6060", nil),
			})
		}()
	}
	reporter := a.newTelemetryReporter()
	reporter.Start(ctx)

	producer := a.newProducer()
	consumer, err := a.newConsumer(ctx)
	if err != nil {
		a.logger.Fatal("construction", err)
	}

	consumer.Start(producer)

	blockTillInterrupt()

	a.logger.Info("app", lager.Data{"cleanup": "exit received, attempting to flush buffers"})
	if err := consumer.Stop(); err != nil {
		a.logger.Error("nozzle.stop", err)
	}
	cancel()

	t := time.NewTimer(5 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ticker.C:
			if a.bufferEmpty() {
				a.logger.Info("app", lager.Data{"cleanup": "The metrics buffer was successfully flushed before shutdown"})
				return
			}
		case <-t.C:
			a.logger.Info("app", lager.Data{"cleanup": "The metrics buffer could not be flushed before shutdown"})
			return
		}
	}
}

func blockTillInterrupt() {
	c := make(chan os.Signal, 1)
	defer close(c)
	signal.Notify(c, os.Interrupt)
	<-c
	signal.Stop(c)
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
		if err := logAdapter.Flush(); err != nil {
			fmt.Printf("error flushing when handling fatal error: %v", err)
		}

		// Re-throw the error, we want to ensure it's logged directly to
		// stackdriver but we are not in a recoverable state.
		panic(e)
	}
}

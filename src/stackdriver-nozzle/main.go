package main

import (
	"context"
	_ "net/http/pprof"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/app"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/config"
)

func main() {
	logger := lager.NewLogger("stackdriver-nozzle")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))
	logger.RegisterSink(lager.NewWriterSink(os.Stderr, lager.ERROR))

	cfg, err := config.NewConfig()
	if err != nil {
		logger.Fatal("config", err)
	}

	a := app.New(cfg, logger)

	ctx := context.Background()
	app.Run(ctx, a)
}

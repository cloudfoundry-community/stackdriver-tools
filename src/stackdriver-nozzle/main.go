package main

import (
	"context"
	_ "net/http/pprof"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/app"
)

func main() {
	ctx := context.Background()
	a := app.New()
	app.Run(ctx, a)
}

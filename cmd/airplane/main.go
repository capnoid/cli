package main

import (
	"context"
	"errors"
	"log"

	"github.com/airplanedev/cli/commands/root"
	"github.com/airplanedev/cli/pkg/trap"
)

var (
	version = "<dev>"
)

func main() {
	var cmd = root.New()
	var ctx = trap.Context()

	cmd.Version = version

	if err := cmd.ExecuteContext(ctx); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Fatalln(err)
		}
	}
}

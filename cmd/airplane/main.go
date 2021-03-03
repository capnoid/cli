package main

import (
	"context"
	"errors"
	"log"

	"github.com/airplanedev/cli/commands/root"
)

var (
	version = "<dev>"
)

func main() {
	var cmd = root.New()

	cmd.Version = version

	if err := cmd.Execute(); err != nil {
		if !errors.Is(err, context.Canceled) {
			log.Fatalln(err)
		}
	}
}

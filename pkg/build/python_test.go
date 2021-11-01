package build

import (
	"context"
	"testing"

	"github.com/airplanedev/cli/pkg/api"
)

func TestPythonBuilder(t *testing.T) {
	ctx := context.Background()

	tests := []Test{
		{
			Root: "python/simple",
			Kind: api.TaskKindPython,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.py",
			},
		},
	}

	RunTests(t, ctx, tests)
}

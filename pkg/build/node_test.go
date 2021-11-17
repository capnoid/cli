package build

import (
	"context"
	"testing"

	"github.com/airplanedev/cli/pkg/api"
)

func TestNodeBuilder(t *testing.T) {
	ctx := context.Background()

	tests := []Test{
		{
			Root: "javascript/simple",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.js",
			},
		},
		{
			Root: "typescript/simple",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
		},
		{
			Root: "typescript/npm",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
		},
		{
			Root: "typescript/yarn",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
		},
		{
			Root: "typescript/imports",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "task/main.ts",
			},
		},
		{
			Root: "typescript/noparams",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
			// Since this example does not take parameters, override the default SearchString.
			SearchString: "success",
		},
		{
			Root: "typescript/esnext",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
		},
		{
			Root: "typescript/esnext",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":        "true",
				"entrypoint":  "main.ts",
				"nodeVersion": "12",
			},
		},
		{
			Root: "typescript/esnext",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":        "true",
				"entrypoint":  "main.ts",
				"nodeVersion": "14",
			},
		},
		{
			Root: "typescript/esm",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
		},
		{
			Root: "typescript/aliases",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
		},
		{
			Root: "typescript/externals",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "main.ts",
			},
		},
		{
			Root: "typescript/yarnworkspaces",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "pkg2/src/index.ts",
			},
		},
		{
			Root: "typescript/nodeworkspaces",
			Kind: api.TaskKindNode,
			Options: api.KindOptions{
				"shim":       "true",
				"entrypoint": "pkg2/src/index.ts",
			},
		},
	}

	RunTests(t, ctx, tests)
}

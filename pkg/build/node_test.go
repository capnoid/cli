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
		// TODO: debug why yarn workspaces aren't working. Seems like we would need to compile
		// pkg1 before compiling pkg2. Once we do that, add an npm workspaces variant along with
		// JS variants.
		// {
		// 	Root: "typescript/yarnworkspaces",
		// 	Kind: api.TaskKindNode,
		// 	Options: api.KindOptions{
		// 		"shim":       "true",
		// 		"entrypoint": "pkg2/src/index.ts",
		// 	},
		// },
	}

	RunTests(t, ctx, tests)
}

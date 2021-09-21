package build

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/stretchr/testify/require"
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

func TestGenTSConfig(t *testing.T) {
	require := require.New(t)

	// No tsconfig.json: should use defaults:
	c, err := GenTSConfig("testdata/tsconfigs/none", "testdata/tsconfigs/none/main.ts", nil)
	require.NoError(err)
	m := map[string]interface{}{}
	require.NoError(json.Unmarshal(c, &m))
	require.Equal(map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"allowJs":         true,
			"esModuleInterop": true,
			"lib":             []interface{}{"esnext", "dom"},
			"module":          "commonjs",
			"outDir":          "./dist",
			"rootDir":         "..",
			"skipLibCheck":    true,
			"target":          "esnext",
		},
		"files": []interface{}{"./shim.ts"},
	}, m)

	// Empty user-provided tsconfig.json: should use defaults:
	c, err = GenTSConfig("testdata/tsconfigs/empty", "testdata/tsconfigs/empty/main.ts", nil)
	require.NoError(err)
	m = map[string]interface{}{}
	require.NoError(json.Unmarshal(c, &m))
	require.Equal(map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"allowJs":         true,
			"esModuleInterop": true,
			"lib":             []interface{}{"esnext", "dom"},
			"module":          "commonjs",
			"outDir":          "./dist",
			"rootDir":         "..",
			"skipLibCheck":    true,
			"target":          "esnext",
		},
		"files":   []interface{}{"./shim.ts"},
		"extends": "../tsconfig.json",
	}, m)

	// Empty user-provided tsconfig.json w/ node 12: should use older lib:
	c, err = GenTSConfig("testdata/tsconfigs/empty", "testdata/tsconfigs/empty/main.ts", api.KindOptions{
		"nodeVersion": "12",
	})
	require.NoError(err)
	m = map[string]interface{}{}
	require.NoError(json.Unmarshal(c, &m))
	require.Equal(map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"allowJs":         true,
			"esModuleInterop": true,
			"lib":             []interface{}{"es2019", "dom"}, // <-
			"module":          "commonjs",
			"outDir":          "./dist",
			"rootDir":         "..",
			"skipLibCheck":    true,
			"target":          "es2019", // <-
		},
		"files":   []interface{}{"./shim.ts"},
		"extends": "../tsconfig.json",
	}, m)

	// Partially filled user-provided tsconfig.json: should accept overrides:
	c, err = GenTSConfig("testdata/tsconfigs/partial", "testdata/tsconfigs/partial/main.ts", nil)
	require.NoError(err)
	m = map[string]interface{}{}
	require.NoError(json.Unmarshal(c, &m))
	require.Equal(map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			"allowJs": true,
			// esModuleInterop omitted
			// lib omitted
			"module":       "commonjs",
			"outDir":       "./dist",
			"rootDir":      "..",
			"skipLibCheck": true,
			// target omitted
		},
		"files":   []interface{}{"./shim.ts"},
		"extends": "../tsconfig.json",
	}, m)

	// Fully specified user-provided tsconfig.json: should accept all:
	c, err = GenTSConfig("testdata/tsconfigs/full", "testdata/tsconfigs/full/main.ts", nil)
	require.NoError(err)
	m = map[string]interface{}{}
	require.NoError(json.Unmarshal(c, &m))
	require.Equal(map[string]interface{}{
		"compilerOptions": map[string]interface{}{
			// allowJs omitted
			// esModuleInterop omitted
			// lib omitted
			// module omitted
			"outDir":  "./dist",
			"rootDir": "..",
			// skipLibCheck omitted
			// target omitted
		},
		"files":   []interface{}{"./shim.ts"},
		"extends": "../tsconfig.json",
	}, m)
}

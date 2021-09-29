package build

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

// ExternalPackages reads package.json and returns all dependencies and dev dependencies.
// This is used as a bit of a workaround for esbuild - we're using esbuild to transform code but
// don't actually want it to bundle. We hit issues when it tries to bundle optional packages
// (and the packages aren't installed) - for example, pg optionally depends on pg-native, and
// using just pg causes esbuild to bundle pg which bundles pg-native, which errors.
// TODO: replace this with a cleaner esbuild plugin that can mark dependencies as external:
// https://github.com/evanw/esbuild/issues/619#issuecomment-751995294
func ExternalPackages(pathPackageJSON string) ([]string, error) {
	var deps []string

	f, err := os.Open(pathPackageJSON)
	if err != nil {
		return nil, errors.Wrap(err, "opening package.json")
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "reading package.json")
	}
	var d struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, errors.Wrap(err, "unmarshaling package.json")
	}

	for k := range d.Dependencies {
		deps = appendDep(deps, k)
	}
	for k := range d.DevDependencies {
		deps = appendDep(deps, k)
	}
	for k := range d.OptionalDependencies {
		deps = appendDep(deps, k)
	}
	return deps, nil
}

var esmModules = []string{
	"node-fetch",
}

func appendDep(deps []string, dep string) []string {
	// For known ESM modules, do not mark as external and use esbuild to bundle.
	// As long as these modules don't happen to pull in any optional modules, we should be OK.
	// This is a bandaid until we figure out how to handle ESM without bundling.
	for _, m := range esmModules {
		if m == dep {
			return deps
		}
	}
	return append(deps, dep)
}

package build

import (
	"context"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	"github.com/airplanedev/lib/pkg/build"
)

// Request represents a build request.
type Request struct {
	Local   bool
	Client  *api.Client
	Root    string
	Def     definitions.DefinitionInterface
	TaskID  string
	TaskEnv api.TaskEnv
	Shim    bool
	GitMeta api.BuildGitMeta
}

// Response represents a build response.
type Response struct {
	ImageURL string
	// Optional, only if applicable
	BuildID string
}

// Run runs the build and returns an image URL.
func Run(ctx context.Context, deployer *Deployer, req Request) (*build.Response, error) {
	if req.Local {
		return deployer.local(ctx, req)
	}
	return deployer.remote(ctx, req)
}

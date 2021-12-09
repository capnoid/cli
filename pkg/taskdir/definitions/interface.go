package definitions

import (
	"context"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/lib/pkg/build"
)

type DefinitionInterface interface {
	GetKindAndOptions() (build.TaskKind, build.KindOptions, error)
	GetEnv() (api.TaskEnv, error)
	GetSlug() string
	UpgradeJST() error
	UpdateTaskRequest(context.Context, api.APIClient, *string) (api.UpdateTaskRequest, error)
}

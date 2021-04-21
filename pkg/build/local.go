package build

import (
	"context"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/taskdir"
	"github.com/pkg/errors"
)

func Local(ctx context.Context, client *api.Client, dir taskdir.TaskDirectory, def taskdir.Definition, taskID string) error {
	registry, err := client.GetRegistryToken(ctx)
	if err != nil {
		return errors.Wrap(err, "getting registry token")
	}

	b, err := New(Config{
		Root:    dir.DefinitionRootPath(),
		Builder: def.Builder,
		Args:    Args(def.BuilderConfig),
		Auth: &RegistryAuth{
			Token: registry.Token,
			Repo:  registry.Repo,
		},
	})
	if err != nil {
		return errors.Wrap(err, "new build")
	}

	logger.Log("Building...")
	bo, err := b.Build(ctx, taskID, "latest")
	if err != nil {
		return errors.Wrap(err, "build")
	}

	logger.Log("Pushing...")
	if err := b.Push(ctx, bo.Tag); err != nil {
		return errors.Wrap(err, "push")
	}

	return nil
}
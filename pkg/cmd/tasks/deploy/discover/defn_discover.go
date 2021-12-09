package discover

import (
	"context"
	"path/filepath"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/taskdir"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
)

type DefnDiscoverer struct {
	Client api.APIClient
}

var _ TaskDiscoverer = &DefnDiscoverer{}

func (dd *DefnDiscoverer) IsAirplaneTask(ctx context.Context, file string) (slug string, err error) {
	dir, err := taskdir.Open(file, true)
	if err != nil {
		return "", nil
	}
	defer dir.Close()

	def, err := dir.ReadDefinition_0_3()
	if err != nil {
		return "", err
	}

	return def.Slug, nil
}

func (dd *DefnDiscoverer) GetTaskConfig(ctx context.Context, task api.Task, file string) (TaskConfig, error) {
	dir, err := taskdir.Open(file, true)
	if err != nil {
		return TaskConfig{}, err
	}
	defer dir.Close()

	def, err := dir.ReadDefinition_0_3()
	if err != nil {
		return TaskConfig{}, err
	}

	utr, err := def.GetUpdateTaskRequest(ctx, dd.Client, nil)
	if err != nil {
		return TaskConfig{}, err
	}

	taskFilePath := ""
	entrypoint, err := def.Entrypoint()
	if err == definitions.ErrNoEntrypoint {
		// nothing
	} else if err != nil {
		return TaskConfig{}, err
	} else {
		taskFilePath, err = filepath.Abs(entrypoint)
		if err != nil {
			return TaskConfig{}, err
		}
	}

	return TaskConfig{
		TaskRoot:     dir.DefinitionRootPath(),
		TaskFilePath: taskFilePath,
		Task:         task,
		Def:          &def,
		Kind:         utr.Kind,
		KindOptions:  utr.KindOptions,
	}, nil
}

func (dd *DefnDiscoverer) TaskConfigSource() TaskConfigSource {
	return TaskConfigSourceDefn
}

package deploy

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/build"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/runtime"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	"github.com/pkg/errors"
)

// DeployFromScript deploys from the given script.
func deployFromScript(ctx context.Context, cfg config) error {
	var client = cfg.client
	var ext = filepath.Ext(cfg.file)

	if ext == "" {
		return errors.New("cannot deploy a file without extension")
	}

	r, ok := runtime.Lookup(cfg.file)
	if !ok {
		return fmt.Errorf("cannot deploy a file with extension of %q", ext)
	}

	code, err := ioutil.ReadFile(cfg.file)
	if err != nil {
		return fmt.Errorf("reading %s - %w", cfg.file, err)
	}

	slug, ok := r.Slug(code)
	if !ok {
		return &unlinked{
			path: cfg.file,
		}
	}

	task, err := client.GetTask(ctx, slug)
	if err != nil {
		return err
	}

	// TODO: make the expected kind a property of the `runtime`
	if task.Kind != api.TaskKindNode {
		return fmt.Errorf("'%s' is a %s task. Expected a %s task.", task.Name, task.Kind, api.TaskKindNode)
	}

	def, err := definitions.NewDefinitionFromTask(task)
	if err != nil {
		return err
	}

	abs, err := filepath.Abs(cfg.file)
	if err != nil {
		return err
	}

	def.Node.Entrypoint = filepath.Base(abs)

	kind, kindOptions, err := def.GetKindAndOptions()
	if err != nil {
		return err
	}

	// Before performing a remote build, we must first update kind/kindOptions
	// since the remote build relies on pulling those from the tasks table (for now).
	_, err = client.UpdateTask(ctx, api.UpdateTaskRequest{
		Kind:        kind,
		KindOptions: kindOptions,

		// The following fields are not updated until after the build finishes.
		Slug:             task.Slug,
		Name:             task.Name,
		Description:      task.Description,
		Image:            task.Image,
		Command:          task.Command,
		Arguments:        task.Arguments,
		Parameters:       task.Parameters,
		Constraints:      task.Constraints,
		Env:              task.Env,
		ResourceRequests: task.ResourceRequests,
		Resources:        task.Resources,
		Repo:             task.Repo,
		Timeout:          task.Timeout,
	})
	if err != nil {
		return errors.Wrapf(err, "updating task %s", def.Slug)
	}

	resp, err := build.Run(ctx, build.Request{
		Local:   cfg.local,
		Client:  client,
		TaskID:  task.ID,
		Root:    filepath.Dir(abs),
		Def:     def,
		TaskEnv: def.Env,
	})
	if err != nil {
		return err
	}

	_, err = client.UpdateTask(ctx, api.UpdateTaskRequest{
		Slug:             def.Slug,
		Name:             def.Name,
		Description:      def.Description,
		Image:            &resp.ImageURL,
		Command:          []string{},
		Arguments:        def.Arguments,
		Parameters:       def.Parameters,
		Constraints:      def.Constraints,
		Env:              def.Env,
		ResourceRequests: def.ResourceRequests,
		Resources:        def.Resources,
		Kind:             kind,
		KindOptions:      kindOptions,
		Repo:             def.Repo,
		Timeout:          def.Timeout,
	})
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("airplane execute %s", cfg.file)
	if len(def.Parameters) > 0 {
		cmd += " -- [parameters]"
	}
	logger.Log(`
To execute %s:
- From the CLI: %s
- From the UI: %s`, def.Name, cmd, client.TaskURL(task.Slug))
	return nil
}

// Unlinked explains an unlinked code error.
type unlinked struct {
	path string
}

// Error implementation.
func (u unlinked) Error() string {
	return fmt.Sprintf(
		"the file %s is not linked to a task",
		u.path,
	)
}

// ExplainError implementation.
func (u unlinked) ExplainError() string {
	return fmt.Sprintf(
		"You can link the file by running:\n\tairplane init --slug <slug> %s",
		u.path,
	)
}
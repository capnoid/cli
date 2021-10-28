package deploy

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/airplanedev/cli/pkg/analytics"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/build"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/runtime"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	"github.com/airplanedev/cli/pkg/utils/pointers"
	"github.com/pkg/errors"
)

// DeployFromScript deploys from the given script.
func deployFromScript(ctx context.Context, cfg config) (rErr error) {
	client := cfg.client
	tp := taskDeployedProps{
		from: "script",
	}
	start := time.Now()
	defer func() {
		analytics.Track(cfg.root, "Task Deployed", map[string]interface{}{
			"from":             tp.from,
			"kind":             tp.kind,
			"task_id":          tp.taskID,
			"task_slug":        tp.taskSlug,
			"task_name":        tp.taskName,
			"build_id":         tp.buildID,
			"errored":          rErr != nil,
			"duration_seconds": time.Since(start).Seconds(),
		})
	}()

	code, err := ioutil.ReadFile(cfg.file)
	if err != nil {
		return errors.Wrapf(err, "reading %s", cfg.file)
	}

	slug, ok := runtime.Slug(code)
	if !ok {
		return runtime.ErrNotLinked{Path: cfg.file}
	}

	task, err := client.GetTask(ctx, slug)
	if err != nil {
		return err
	}
	tp.kind = task.Kind
	tp.taskID = task.ID
	tp.taskSlug = task.Slug
	tp.taskName = task.Name

	r, err := runtime.Lookup(cfg.file, task.Kind)
	if err != nil {
		return errors.Wrapf(err, "cannot determine how to deploy %q - check your CLI is up to date", cfg.file)
	}

	def, err := definitions.NewDefinitionFromTask(task)
	if err != nil {
		return err
	}

	absFile, err := filepath.Abs(cfg.file)
	if err != nil {
		return err
	}

	taskroot, err := r.Root(absFile)
	if err != nil {
		return err
	}
	if err := def.SetEntrypoint(taskroot, absFile); err != nil {
		return err
	}

	wd, err := r.Workdir(absFile)
	if err != nil {
		return err
	}
	def.SetWorkdir(taskroot, wd)

	kind, kindOptions, err := def.GetKindAndOptions()
	if err != nil {
		return err
	}

	interpolationMode := task.InterpolationMode
	if interpolationMode != "jst" {
		if cfg.upgradeInterpolation {
			logger.Warning(`Your task is being migrated from handlebars to Airplane JS Templates.
More information: https://apn.sh/jst-upgrade`)
			interpolationMode = "jst"
			def.UpgradeJST()
		} else {
			logger.Warning(`Tasks are migrating from handlebars to Airplane JS Templates! Your task has not
been automatically upgraded because of potential backwards-compatibility issues
(e.g. uploads will be passed to your task as an object with a url field instead
of just the url string).

To upgrade, update your task to support the new format and re-deploy with --jst.
More information: https://apn.sh/jst-upgrade`)
		}
	}

	tp.buildLocal = cfg.local
	resp, err := build.Run(ctx, build.Request{
		Local:   cfg.local,
		Client:  client,
		TaskID:  task.ID,
		Root:    taskroot,
		Def:     def,
		TaskEnv: def.Env,
		Shim:    true,
	})
	if err != nil {
		return err
	}
	tp.buildID = resp.BuildID

	_, err = client.UpdateTask(ctx, api.UpdateTaskRequest{
		Slug:                       def.Slug,
		Name:                       def.Name,
		Description:                def.Description,
		Image:                      &resp.ImageURL,
		Command:                    []string{},
		Arguments:                  def.Arguments,
		Parameters:                 def.Parameters,
		Constraints:                def.Constraints,
		Env:                        def.Env,
		ResourceRequests:           def.ResourceRequests,
		Resources:                  def.Resources,
		Kind:                       kind,
		KindOptions:                kindOptions,
		Repo:                       def.Repo,
		RequireExplicitPermissions: task.RequireExplicitPermissions,
		Permissions:                task.Permissions,
		Timeout:                    def.Timeout,
		BuildID:                    pointers.String(resp.BuildID),
		InterpolationMode:          interpolationMode,
	})
	if err != nil {
		return err
	}

	// Leave off `-- [parameters]` for simplicity - user will get prompted.
	cmd := fmt.Sprintf("airplane exec %s", cfg.file)
	logger.Suggest(
		"⚡ To execute the task from the CLI:",
		cmd,
	)

	logger.Suggest(
		"⚡ To execute the task from the UI:",
		client.TaskURL(task.Slug),
	)
	return nil
}

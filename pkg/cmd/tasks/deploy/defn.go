package deploy

import (
	"context"
	"path/filepath"
	"time"

	"github.com/airplanedev/cli/pkg/analytics"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/build"
	"github.com/airplanedev/cli/pkg/conf"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/taskdir"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	"github.com/airplanedev/cli/pkg/utils/pointers"
	libBuild "github.com/airplanedev/lib/pkg/build"
	"github.com/pkg/errors"
)

// deployFromTaskDefn deploys from a task definition file.
func deployFromTaskDefn(ctx context.Context, cfg config) error {
	dir, err := taskdir.Open(cfg.paths[0], true)
	if err != nil {
		return err
	}
	defer dir.Close()

	def, err := dir.ReadDefinition_0_3()
	if err != nil {
		return err
	}

	task, err := cfg.client.GetTask(ctx, def.Slug)
	if _, ok := err.(*api.TaskMissingError); ok {
		// TODO: confirm with user & create the task.
		return errors.New("NotImplemented: new task")
	} else if err != nil {
		return errors.Wrap(err, "getting task")
	}

	tc, err := getTaskConfigFromDefn(ctx, *cfg.client, def, task, dir.DefinitionRootPath())
	if err != nil {
		return err
	}

	if err := deploySingleTaskFromTaskDefn(ctx, cfg, tc); err != nil {
		logger.Log("\n" + logger.Bold(tc.def.GetSlug()))
		logger.Log("Status: " + logger.Bold(logger.Red("failed")))
		logger.Error(err.Error())
		return err
	}
	logger.Log("\n" + logger.Bold(tc.def.GetSlug()))
	logger.Log("Status: %s", logger.Bold(logger.Green("succeeded")))
	logger.Log("Execute the task: %s", cfg.client.TaskURL(tc.def.GetSlug()))
	return nil
}

func deploySingleTaskFromTaskDefn(ctx context.Context, cfg config, tc taskConfig) (rErr error) {
	client := cfg.client
	props := taskDeployedProps{
		from: "defn",
	}
	start := time.Now()
	defer func() {
		analytics.Track(cfg.root, "Task Deployed", map[string]interface{}{
			"from":             props.from,
			"kind":             props.kind,
			"task_id":          props.taskID,
			"task_slug":        props.taskSlug,
			"task_name":        props.taskName,
			"build_id":         props.buildID,
			"errored":          rErr != nil,
			"duration_seconds": time.Since(start).Seconds(),
		})
	}()

	task := tc.task

	props.taskSlug = tc.def.GetSlug()
	props.taskID = task.ID
	props.taskName = task.Name

	logger.Log(logger.Bold(tc.task.Slug))
	logger.Log("Type: %s", tc.kind)
	logger.Log("Root directory: %s", relpath(tc.taskRoot))
	if tc.workingDirectory != tc.taskRoot {
		logger.Log("Working directory: %s", relpath(tc.workingDirectory))
	}
	logger.Log("URL: %s", cfg.client.TaskURL(tc.task.Slug))
	logger.Log("")

	interpolationMode := task.InterpolationMode
	if interpolationMode != "jst" {
		if cfg.upgradeInterpolation {
			logger.Warning(`Your task is being migrated from handlebars to Airplane JS Templates.
More information: https://apn.sh/jst-upgrade`)
			interpolationMode = "jst"
			if err := tc.def.UpgradeJST(); err != nil {
				return err
			}
		} else {
			logger.Warning(`Tasks are migrating from handlebars to Airplane JS Templates! Your task has not
been automatically upgraded because of potential backwards-compatibility issues
(e.g. uploads will be passed to your task as an object with a url field instead
of just the url string).

To upgrade, update your task to support the new format and re-deploy with --jst.
More information: https://apn.sh/jst-upgrade`)
		}
	}

	gitMeta, err := getGitMetadata(tc.taskFilePath)
	if err != nil {
		logger.Debug("failed to gather git metadata: %v", err)
		analytics.ReportError(errors.Wrap(err, "failed to gather git metadata"))
	}
	gitMeta.User = conf.GetGitUser()
	gitMeta.Repository = conf.GetGitRepo()

	var image *string
	var buildID string
	kind, _, err := tc.def.GetKindAndOptions()
	if err != nil {
		return err
	}
	if ok, err := libBuild.NeedsBuilding(kind); err != nil {
		return err
	} else if ok {
		resp, err := build.Run(ctx, build.NewDeployer(), build.Request{
			Local:   cfg.local,
			Client:  client,
			TaskID:  task.ID,
			Root:    tc.taskRoot,
			Def:     tc.def,
			Shim:    true,
			GitMeta: gitMeta,
		})
		props.buildLocal = cfg.local
		if resp != nil {
			props.buildID = resp.BuildID
			buildID = resp.BuildID
		}
		if err != nil {
			return err
		}
		image = &resp.ImageURL
	}

	updateTaskRequest, err := tc.def.UpdateTaskRequest(ctx, client, image)
	if err != nil {
		return err
	}

	updateTaskRequest.BuildID = pointers.String(buildID)
	updateTaskRequest.InterpolationMode = interpolationMode

	if _, err = client.UpdateTask(ctx, updateTaskRequest); err != nil {
		return errors.Wrapf(err, "updating task %s", tc.def.GetSlug())
	}
	return nil
}

func getTaskConfigFromDefn(ctx context.Context, client api.Client, def definitions.Definition_0_3, task api.Task, root string) (taskConfig, error) {
	utr, err := def.UpdateTaskRequest(ctx, &client, nil)
	if err != nil {
		return taskConfig{}, err
	}

	taskFilePath := ""
	entrypoint, err := def.Entrypoint()
	if err == definitions.ErrNoEntrypoint {
		// nothing
	} else if err != nil {
		return taskConfig{}, err
	} else {
		taskFilePath, err = filepath.Abs(entrypoint)
		if err != nil {
			return taskConfig{}, err
		}
	}

	return taskConfig{
		taskRoot:     root,
		taskFilePath: taskFilePath,
		task:         task,
		def:          &def,
		kind:         utr.Kind,
		kindOptions:  utr.KindOptions,
	}, nil
}

package deploy

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/airplanedev/cli/pkg/analytics"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/build"
	"github.com/airplanedev/cli/pkg/conf"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	"github.com/airplanedev/cli/pkg/utils/pointers"
	libBuild "github.com/airplanedev/lib/pkg/build"
	"github.com/airplanedev/lib/pkg/runtime"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var ignoredDirectories = map[string]bool{
	"node_modules": true,
	"__pycache__":  true,
	".git":         true,
}

type scriptDeployer struct {
	deployer *build.Deployer

	erroredTaskSlugs  map[string]error
	deployedTaskSlugs []string
	mu                sync.Mutex
}

func NewDeployer() *scriptDeployer {
	return &scriptDeployer{
		deployer:         build.NewDeployer(),
		erroredTaskSlugs: make(map[string]error),
	}
}

// deployFromScript deploys N tasks from the given set of files or directories.
func (d *scriptDeployer) deployFromScript(ctx context.Context, cfg config) error {
	loader := logger.NewLoader(logger.LoaderOpts{HideLoader: logger.EnableDebug})
	loader.Start()
	scriptsToDeploy, err := d.discoverScripts(ctx, cfg.paths...)
	if err != nil {
		return err
	}
	loader.Stop()

	var taskConfigs []taskConfig
	for _, script := range scriptsToDeploy {
		tc, err := getTaskConfigFromScript(ctx, *cfg.client, script)
		if err != nil {
			return err
		}
		taskConfigs = append(taskConfigs, tc)
	}

	if len(cfg.changedFiles) > 0 {
		// Filter out any tasks that don't have changed files.
		var filteredTaskConfigs []taskConfig
		for _, tc := range taskConfigs {
			contains, err := containsFile(tc.taskRoot, cfg.changedFiles)
			if err != nil {
				return err
			}
			if contains {
				filteredTaskConfigs = append(filteredTaskConfigs, tc)
			}
		}
		if len(taskConfigs) != len(filteredTaskConfigs) {
			logger.Log("Changed files specified. Filtered %d task(s) to %d affected task(s)", len(taskConfigs), len(filteredTaskConfigs))
		}
		taskConfigs = filteredTaskConfigs
	}

	if len(taskConfigs) == 0 {
		logger.Log("No tasks to deploy")
		return nil
	}

	// Print out a summary before deploying.
	noun := "task"
	if len(taskConfigs) > 1 {
		noun = fmt.Sprintf("%ss", noun)
	}
	logger.Log("Deploying %v %v:\n", len(taskConfigs), noun)
	for _, tc := range taskConfigs {
		logger.Log(logger.Bold(tc.task.Slug))
		logger.Log("Type: %s", tc.task.Kind)
		logger.Log("Root directory: %s", relpath(tc.taskRoot))
		if tc.workingDirectory != tc.taskRoot {
			logger.Log("Working directory: %s", relpath(tc.workingDirectory))
		}
		logger.Log("URL: %s", cfg.client.TaskURL(tc.task.Slug))
		logger.Log("")
	}

	g := new(errgroup.Group)
	// Concurrently deploy the tasks.
	for _, tc := range taskConfigs {
		tc := tc
		g.Go(func() error {
			err := d.deploySingleTaskFromScript(ctx, cfg, tc)
			d.mu.Lock()
			defer d.mu.Unlock()
			if err != nil {
				if !errors.As(err, &runtime.ErrNotLinked{}) {
					d.erroredTaskSlugs[tc.task.Slug] = err
					return err
				}
			} else {
				d.deployedTaskSlugs = append(d.deployedTaskSlugs, tc.task.Slug)
			}
			return nil
		})
	}

	groupErr := g.Wait()

	// All of the deploys have finished.
	for taskSlug, err := range d.erroredTaskSlugs {
		logger.Log("\n" + logger.Bold(taskSlug))
		logger.Log("Status: " + logger.Bold(logger.Red("failed")))
		logger.Error(err.Error())
	}
	for _, slug := range d.deployedTaskSlugs {
		logger.Log("\n" + logger.Bold(slug))
		logger.Log("Status: %s", logger.Bold(logger.Green("succeeded")))
		logger.Log("Execute the task: %s", cfg.client.TaskURL(slug))
	}

	return groupErr
}

type script struct {
	file     string
	taskSlug string
}

// discoverScripts recursively discovers Airplane task scripts.
func (d *scriptDeployer) discoverScripts(ctx context.Context, paths ...string) ([]script, error) {
	var scripts []script
	for _, p := range paths {
		if ignoredDirectories[p] {
			continue
		}
		logger.Debug("Exploring file or directory: %s", p)
		fileInfo, err := os.Stat(p)
		if err != nil {
			return nil, errors.Wrapf(err, "determining if %s is file or directory", p)
		}

		if fileInfo.IsDir() {
			// We found a directory. Recursively explore all of the files and directories in it.
			nestedFiles, err := ioutil.ReadDir(p)
			if err != nil {
				return nil, errors.Wrapf(err, "reading directory %s", p)
			}
			var nestedPaths []string
			for _, nestedFile := range nestedFiles {
				nestedPaths = append(nestedPaths, path.Join(p, nestedFile.Name()))
			}
			nestedScripts, err := d.discoverScripts(ctx, nestedPaths...)
			if err != nil {
				return nil, err
			}
			scripts = append(scripts, nestedScripts...)
			continue
		}
		// We found a file.
		slug, ok := runtime.Slug(p)
		if !ok {
			// File is not an Airplane script.
			continue
		}

		scripts = append(scripts, script{
			file:     p,
			taskSlug: slug,
		})
	}

	return scripts, nil
}

func (d *scriptDeployer) deploySingleTaskFromScript(ctx context.Context, cfg config, tc taskConfig) (rErr error) {
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

	task := tc.task

	tp.kind = tc.kind
	tp.taskID = task.ID
	tp.taskSlug = task.Slug
	tp.taskName = task.Name
	tp.buildLocal = cfg.local

	interpolationMode := task.InterpolationMode
	if interpolationMode != "jst" {
		if cfg.upgradeInterpolation {
			logger.Warning(`Your task is being migrated from handlebars to Airplane JS Templates.
More information: https://apn.sh/jst-upgrade`)
			interpolationMode = "jst"
			tc.def.UpgradeJST()
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

	resp, err := build.Run(ctx, d.deployer, build.Request{
		Local:   cfg.local,
		Client:  client,
		TaskID:  task.ID,
		Root:    tc.taskRoot,
		Def:     tc.def,
		TaskEnv: tc.def.Env,
		Shim:    true,
		GitMeta: gitMeta,
	})
	if err != nil {
		return err
	}
	tp.buildID = resp.BuildID

	_, err = client.UpdateTask(ctx, api.UpdateTaskRequest{
		Slug:                       tc.def.Slug,
		Name:                       tc.def.Name,
		Description:                tc.def.Description,
		Image:                      &resp.ImageURL,
		Command:                    []string{},
		Arguments:                  tc.def.Arguments,
		Parameters:                 tc.def.Parameters,
		Constraints:                tc.def.Constraints,
		Env:                        tc.def.Env,
		ResourceRequests:           tc.def.ResourceRequests,
		Resources:                  tc.def.Resources,
		Kind:                       tc.kind,
		KindOptions:                tc.kindOptions,
		Repo:                       tc.def.Repo,
		RequireExplicitPermissions: task.RequireExplicitPermissions,
		Permissions:                task.Permissions,
		Timeout:                    tc.def.Timeout,
		BuildID:                    pointers.String(resp.BuildID),
		InterpolationMode:          interpolationMode,
	})
	return err
}

type taskConfig struct {
	taskRoot         string
	workingDirectory string
	taskFilePath     string
	task             api.Task
	def              definitions.Definition
	kind             libBuild.TaskKind
	kindOptions      libBuild.KindOptions
}

// getTaskConfig a task and associated information from a script.
func getTaskConfigFromScript(ctx context.Context, client api.Client, script script) (taskConfig, error) {
	task, err := client.GetTask(ctx, script.taskSlug)
	if err != nil {
		return taskConfig{}, err
	}

	r, err := runtime.Lookup(script.file, task.Kind)
	if err != nil {
		return taskConfig{}, errors.Wrapf(err, "cannot determine how to deploy %q - check your CLI is up to date", script.file)
	}

	def, err := definitions.NewDefinitionFromTask(task)
	if err != nil {
		return taskConfig{}, err
	}

	absFile, err := filepath.Abs(script.file)
	if err != nil {
		return taskConfig{}, err
	}

	taskroot, err := r.Root(absFile)
	if err != nil {
		return taskConfig{}, err
	}
	if err := def.SetEntrypoint(taskroot, absFile); err != nil {
		return taskConfig{}, err
	}

	wd, err := r.Workdir(absFile)
	if err != nil {
		return taskConfig{}, err
	}
	def.SetWorkdir(taskroot, wd)

	kind, kindOptions, err := def.GetKindAndOptions()
	if err != nil {
		return taskConfig{}, err
	}

	return taskConfig{
		taskRoot:         taskroot,
		workingDirectory: wd,
		taskFilePath:     absFile,
		def:              def,
		kind:             kind,
		kindOptions:      kindOptions,
		task:             task,
	}, nil
}

// containsFile returns true if the directory contains at least one of the files.
func containsFile(dir string, filePaths []string) (bool, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false, errors.Wrapf(err, "calculating absolute path of directory %s", dir)
	}
	for _, cf := range filePaths {
		absCF, err := filepath.Abs(cf)
		if err != nil {
			return false, errors.Wrapf(err, "calculating absolute path of file %s", cf)
		}
		changedFileDir := filepath.Dir(absCF)
		if strings.HasPrefix(changedFileDir, absDir) {
			return true, nil
		}
	}
	return false, nil
}

func getGitMetadata(taskFilePath string) (api.BuildGitMeta, error) {
	meta := api.BuildGitMeta{}

	repo, err := git.PlainOpenWithOptions(filepath.Dir(taskFilePath), &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		if errors.Is(err, git.ErrRepositoryNotExists) {
			return meta, nil
		}
		return meta, err
	}

	w, err := repo.Worktree()
	if err != nil {
		return meta, err
	}
	pathRelativeToGitRoot, err := filepath.Rel(w.Filesystem.Root(), taskFilePath)
	if err != nil {
		return meta, err
	}
	meta.FilePath = pathRelativeToGitRoot

	status, err := w.Status()
	if err != nil {
		return meta, err
	}
	meta.IsDirty = !status.IsClean()

	h, err := repo.Head()
	if err != nil {
		return meta, err
	}

	commit, err := repo.CommitObject(h.Hash())
	if err != nil {
		return meta, err
	}
	meta.CommitHash = commit.Hash.String()
	meta.CommitMessage = commit.Message
	if meta.User != "" {
		meta.User = commit.Author.Name
	}

	ref := h.Name().String()
	if h.Name().IsBranch() {
		ref = strings.TrimPrefix(ref, "refs/heads/")
	}
	meta.Ref = ref

	return meta, nil
}

// Relpath returns the relative using root and the cwd.
func relpath(root string) string {
	if path, err := os.Getwd(); err == nil {
		if rp, err := filepath.Rel(path, root); err == nil {
			if len(rp) == 0 || rp == "." {
				// "." can be missed easily, change it to ./
				return "./"
			}
			return "./" + rp
		}
	}
	return root
}

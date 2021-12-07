package deploy

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/build"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/runtime"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	libBuild "github.com/airplanedev/lib/pkg/build"
	"github.com/pkg/errors"
)

var ignoredDirectories = map[string]bool{
	"node_modules": true,
	"__pycache__":  true,
	".git":         true,
}

type deployer struct {
	deployer *build.Deployer

	erroredTaskSlugs  map[string]error
	deployedTaskSlugs []string
	mu                sync.Mutex
}

func newDeployer() *deployer {
	return &deployer{
		deployer:          build.NewDeployer(),
		deployedTaskSlugs: make([]string),
		erroredTaskSlugs:  make(map[string]error),
	}
}

func (d deployer) deploy(ctx context.Context, cfg config) error {
	loader := logger.NewLoader(logger.LoaderOpts{HideLoader: logger.EnableDebug})
	loader.Start()
	taskConfigs, err := d.discover(ctx, cfg.paths...)
	if err != nil {
		return err
	}
	loader.Stop()
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

// discover recursively discovers Airplane tasks.
func (d deployer) discover(ctx context.Context, paths ...string) ([]taskConfig, error) {
	var taskConfigs []taskConfig
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
			tcs, err := d.discoverScripts(ctx, nestedPaths...)
			if err != nil {
				return nil, err
			}
			taskConfigs = append(taskConfigs, tcs...)
			continue
		}

		// We found a file.
		if slug, ok := runtime.Slug(p); ok {
			// File is an Airplane script.
			taskConfig, err := getTaskConfigFromScript(ctx, *cfg.client, p, slug)
			if err != nil {
				return err
			}
			taskConfigs = append(taskConfigs, taskConfig)
		} else {
			// File is not an Airplane script. Maybe it's a YAML file?
			ext := filepath.Ext(p)
			if ext == ".yml" || ext == ".yaml" {
				taskConfig, err := getTaskConfigFromYAML(ctx, *cfg.client, p)
			}
			taskConfigs = append(taskConfigs, taskConfig)
		}
	}
	return taskConfigs, nil
}

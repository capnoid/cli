package execute

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/airplanedev/cli/pkg/analytics"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/cli"
	"github.com/airplanedev/cli/pkg/cmd/auth/login"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/params"
	"github.com/airplanedev/cli/pkg/print"
	"github.com/airplanedev/cli/pkg/runtime"
	"github.com/airplanedev/cli/pkg/taskdir"
	"github.com/airplanedev/cli/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Config is the execute config.
type config struct {
	root *cli.Config
	// task reference could be a script file, yaml definition or a slug.
	task string
	args []string
}

// New returns a new execute cobra command.
func New(c *cli.Config) *cobra.Command {
	var cfg = config{root: c}

	cmd := &cobra.Command{
		Use:     "execute <slug>",
		Short:   "Execute a task",
		Aliases: []string{"exec"},
		Long:    "Execute a task from the CLI, optionally with specific parameters.",
		Example: heredoc.Doc(`
			airplane execute ./task.js [-- <parameters...>]
			airplane execute hello_world [-- <parameters...>]
			airplane execute ./airplane.yml [-- <parameters...>]
		`),
		PersistentPreRunE: utils.WithParentPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return login.EnsureLoggedIn(cmd.Root().Context(), c)
		}),
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.task != "" {
				// A file was provided with the -f flag. This is deprecated.
				logger.Warning(`The --file/-f flag is deprecated and will be removed in a future release. File paths should be passed as a positional argument instead: airplane execute %s`, cfg.task)
				cfg.args = args
			} else if len(args) > 0 {
				cfg.task = args[0]
				cfg.args = args[1:]
			} else {
				return errors.New("expected 1 argument: airplane execute [./path/to/file | task slug]")
			}

			return run(cmd.Root().Context(), cfg)
		},
	}

	cmd.Flags().StringVarP(&cfg.task, "file", "f", "", "File to deploy (.yaml, .yml, .js, .ts)")
	cli.Must(cmd.Flags().MarkHidden("file")) // --file is deprecated

	return cmd
}

// Run runs the execute command.
func run(ctx context.Context, cfg config) error {
	var client = cfg.root.Client

	var slug string
	var err error
	if f, err := os.Stat(cfg.task); errors.Is(err, os.ErrNotExist) || f.IsDir() {
		// Not a file, assume it's a slug.
		slug = cfg.task
	} else {
		// It's a file, look up the slug form the file.
		slug, err = slugFrom(cfg.task)
		if err != nil {
			return err
		}
	}
	task, err := client.GetTask(ctx, slug)
	if err != nil {
		return err
	}

	if task.Image == nil {
		return &notDeployedError{
			task: cfg.task,
		}
	}

	req := api.RunTaskRequest{
		TaskID:      task.ID,
		ParamValues: make(api.Values),
	}

	logger.Log("Executing %s task: %s", logger.Bold(task.Name), logger.Gray(client.TaskURL(task.Slug)))

	req.ParamValues, err = params.CLI(cfg.args, client, task)
	if errors.Is(err, flag.ErrHelp) {
		return nil
	} else if err != nil {
		return err
	}

	w, err := client.Watcher(ctx, req)
	if err != nil {
		return err
	}

	logger.Log(logger.Gray("Queued run: %s", client.RunURL(w.RunID())))

	var state api.RunState
	agentPrefix := "[agent]"

	for {
		if state = w.Next(); state.Err() != nil {
			break
		}

		for _, l := range state.Logs {
			var loggedText string
			if strings.HasPrefix(l.Text, agentPrefix) {
				// De-emphasize agent logs and remove prefix
				loggedText = logger.Gray(strings.TrimLeft(strings.TrimPrefix(l.Text, agentPrefix), " "))
			} else {
				// Try to leave user logs alone, so they can apply their own colors
				loggedText = fmt.Sprintf("[%s] %s", logger.Gray("log"), l.Text)
			}
			logger.Log(loggedText)
		}

		if state.Stopped() {
			break
		}
	}

	if err := state.Err(); err != nil {
		return err
	}

	print.Outputs(state.Outputs)

	analytics.Track(cfg.root, "Run Executed", map[string]interface{}{
		"task_id":   task.ID,
		"task_name": task.Name,
		"status":    state.Status,
	})

	switch state.Status {
	case api.RunFailed:
		return errors.New("Run has failed")
	}
	return nil
}

// SlugFrom returns the slug from the given file.
func slugFrom(file string) (string, error) {
	switch ext := filepath.Ext(file); ext {
	case ".yml", ".yaml":
		return slugFromYaml(file)
	default:
		return slugFromScript(file)
	}
}

// slugFromYaml attempts to extract a slug from a yaml definition.
func slugFromYaml(file string) (string, error) {
	dir, err := taskdir.Open(file)
	if err != nil {
		return "", err
	}
	defer dir.Close()

	def, err := dir.ReadDefinition()
	if err != nil {
		return "", err
	}

	if def.Slug == "" {
		return "", errors.Errorf("no task slug found in task definition at %s", file)
	}

	return def.Slug, nil
}

// slugFromScript attempts to extract a slug from a script.
func slugFromScript(file string) (string, error) {
	slug, ok := runtime.Slug(file)
	if !ok {
		return "", runtime.ErrNotLinked{Path: file}
	}

	return slug, nil
}

type notDeployedError struct {
	task string
}

// Error implementation.
func (err notDeployedError) Error() string {
	return fmt.Sprintf("task %s was not deployed", err.task)
}

// ExplainError implementation.
func (err notDeployedError) ExplainError() string {
	return fmt.Sprintf("to deploy the task:\n  airplane deploy %s", err.task)
}

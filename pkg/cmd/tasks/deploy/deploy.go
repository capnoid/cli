package deploy

import (
	"context"
	"path/filepath"

	"github.com/MakeNowJust/heredoc"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/cli"
	"github.com/airplanedev/cli/pkg/cmd/auth/login"
	"github.com/airplanedev/cli/pkg/utils"
	"github.com/airplanedev/cli/pkg/version"
	"github.com/airplanedev/lib/pkg/build"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type gitConfig struct {
	commitHash string
	branch     string
	user       string
	repository string
}
type config struct {
	root   *cli.Config
	client *api.Client
	paths  []string
	local  bool

	upgradeInterpolation bool
	git                  gitConfig
}

func New(c *cli.Config) *cobra.Command {
	var cfg = config{
		root:   c,
		client: c.Client,
	}

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a task",
		Long:  "Deploy code from a local directory to Airplane.",
		Example: heredoc.Doc(`
			airplane tasks deploy ./task.ts
			airplane tasks deploy --local ./task.js
			airplane tasks deploy ./my-task.yml
			airplane tasks deploy my-directory
			airplane tasks deploy ./my-task1.yml ./my-task2.yml
		`),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cfg.paths = args
			} else {
				return errors.New("expected 1 argument: airplane deploy ./path/to/file")
			}
			return run(cmd.Root().Context(), cfg)
		},
		PersistentPreRunE: utils.WithParentPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return login.EnsureLoggedIn(cmd.Root().Context(), c)
		}),
	}

	cmd.Flags().BoolVarP(&cfg.local, "local", "L", false, "use a local Docker daemon (instead of an Airplane-hosted builder)")
	cmd.Flags().BoolVar(&cfg.upgradeInterpolation, "jst", false, "Upgrade interpolation to JST")
	cmd.Flags().StringVar(&cfg.git.commitHash, "commitHash", "", "The commit hash of the source code of the deployed task")
	cmd.Flags().StringVar(&cfg.git.branch, "branch", "", "The branch containing the source code of the deployed task")
	cmd.Flags().StringVar(&cfg.git.repository, "repository", "", "The repository containing the source code of the deployed task")
	cmd.Flags().StringVar(&cfg.git.user, "gitUser", "", "The git user who deployed the task")
	cli.Must(cmd.Flags().MarkHidden("commitHash")) // internal use only
	cli.Must(cmd.Flags().MarkHidden("branch"))     // internal use only
	cli.Must(cmd.Flags().MarkHidden("repository")) // internal use only
	cli.Must(cmd.Flags().MarkHidden("gitUser"))    // internal use only

	return cmd
}

// Set of properties to track when deploying
type taskDeployedProps struct {
	from       string
	kind       build.TaskKind
	taskID     string
	taskSlug   string
	taskName   string
	buildLocal bool
	buildID    string
}

func run(ctx context.Context, cfg config) error {
	if err := version.CheckLatest(ctx); err != nil {
		return err
	}

	ext := filepath.Ext(cfg.paths[0])

	if ext == ".yml" || ext == ".yaml" {
		return deployFromYaml(ctx, cfg)
	}

	return NewDeployer().deployFromScript(ctx, cfg)
}

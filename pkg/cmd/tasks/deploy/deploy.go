package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/cli"
	"github.com/airplanedev/cli/pkg/cmd/auth/login"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/utils"
	"github.com/airplanedev/cli/pkg/version"
	"github.com/airplanedev/lib/pkg/build"
	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type gitMeta struct {
	commitHash    string
	commitMessage string
	ref           string
	user          string
	repository    string
	isDirty       bool
}
type config struct {
	root   *cli.Config
	client *api.Client
	paths  []string
	local  bool

	upgradeInterpolation bool
	gitMeta              api.BuildGitMeta
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
			err := addGitMeta(&cfg.gitMeta)
			if err != nil {
				logger.Debug("failed to gather git metadata: %v", err)
			}
			return run(cmd.Root().Context(), cfg)
		},
		PersistentPreRunE: utils.WithParentPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return login.EnsureLoggedIn(cmd.Root().Context(), c)
		}),
	}

	cmd.Flags().BoolVarP(&cfg.local, "local", "L", false, "use a local Docker daemon (instead of an Airplane-hosted builder)")
	cmd.Flags().BoolVar(&cfg.upgradeInterpolation, "jst", false, "Upgrade interpolation to JST")
	cmd.Flags().StringVar(&cfg.gitMeta.Repository, "repository", "", "The repository containing the source code of the deployed task")
	cmd.Flags().StringVar(&cfg.gitMeta.User, "gitUser", "", "The git user who deployed the task")
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

func addGitMeta(meta *api.BuildGitMeta) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	repo, err := git.PlainOpen(dir)
	if err != nil {
		return err
	}

	h, err := repo.Head()
	if err != nil {
		return err
	}
	commit, err := repo.CommitObject(h.Hash())
	if err != nil {
		return err
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

	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	status, err := w.Status()
	if err != nil {
		return err
	}
	meta.IsDirty = !status.IsClean()

	return nil
}

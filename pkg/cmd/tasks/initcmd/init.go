// initcmd defines the implementation of the `airplane tasks init` command.
//
// Even though the command is called "init", we can't name the package "init"
// since that conflicts with the Go init function.
package initcmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/cli"
	"github.com/airplanedev/cli/pkg/cmd/auth/login"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/runtime"
	_ "github.com/airplanedev/cli/pkg/runtime/javascript"
	_ "github.com/airplanedev/cli/pkg/runtime/python"
	_ "github.com/airplanedev/cli/pkg/runtime/shell"
	_ "github.com/airplanedev/cli/pkg/runtime/typescript"
	"github.com/airplanedev/cli/pkg/utils"
	"github.com/airplanedev/lib/pkg/utils/fsx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type config struct {
	client *api.Client
	file   string
	slug   string

	dev       bool
	codeOnly  bool
	defFormat string
	assumeYes bool
	assumeNo  bool
}

func New(c *cli.Config) *cobra.Command {
	var cfg = config{client: c.Client}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a task definition",
		Example: heredoc.Doc(`
			$ airplane tasks init --slug task-slug
			$ airplane tasks init --slug task-slug ./my/task.js
			$ airplane tasks init --slug task-slug ./my/task.ts
		`),
		Args: cobra.MaximumNArgs(1),
		PersistentPreRunE: utils.WithParentPersistentPreRunE(func(cmd *cobra.Command, args []string) error {
			return login.EnsureLoggedIn(cmd.Root().Context(), c)
		}),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				cfg.file = args[0]
			}
			return run(cmd.Root().Context(), cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.slug, "slug", "", "Slug of an existing task to generate from.")

	// Remove dev flag + unhide these flags + deprecate `slug` before release!
	cmd.Flags().BoolVar(&cfg.dev, "dev", false, "Dev mode: warning, not guaranteed to work and subject to change.")
	cmd.Flags().StringVar(&cfg.slug, "from", "", "Slug of an existing task to initialize.")
	cmd.Flags().BoolVar(&cfg.codeOnly, "code-only", false, "True to skip creating a task definition file; only generates an entrypoint file.")
	cmd.Flags().StringVar(&cfg.defFormat, "def-format", "", `One of "json" or "yaml". Defaults to "yaml".`)
	cmd.Flags().BoolVarP(&cfg.assumeYes, "yes", "y", false, "True to specify automatic yes to prompts.")
	cmd.Flags().BoolVarP(&cfg.assumeNo, "no", "n", false, "True to specify automatic no to prompts.")

	if err := cmd.Flags().MarkHidden("dev"); err != nil {
		logger.Debug("error: %s", err)
	}
	if err := cmd.Flags().MarkHidden("from"); err != nil {
		logger.Debug("error: %s", err)
	}
	if err := cmd.Flags().MarkHidden("code-only"); err != nil {
		logger.Debug("error: %s", err)
	}
	if err := cmd.Flags().MarkHidden("def-format"); err != nil {
		logger.Debug("error: %s", err)
	}
	if err := cmd.Flags().MarkHidden("yes"); err != nil {
		logger.Debug("error: %s", err)
	}
	if err := cmd.Flags().MarkHidden("no"); err != nil {
		logger.Debug("error: %s", err)
	}

	return cmd
}

func run(ctx context.Context, cfg config) error {
	if !cfg.dev {
		return initCodeOnly(ctx, cfg)
	}

	// Check for mutually exclusive flags.
	if cfg.codeOnly && cfg.defFormat != "" {
		return errors.New("Cannot specify both --code-only and --def-format")
	}
	if cfg.assumeYes && cfg.assumeNo {
		return errors.New("Cannot specify both --yes and --no")
	}

	if cfg.codeOnly {
		return initCodeOnly(ctx, cfg)
	}

	return initWithTaskDef(ctx, cfg)
}

func initWithTaskDef(ctx context.Context, cfg config) error {
	// Check for a valid defFormat, add in a default if necessary.
	if cfg.defFormat == "" {
		cfg.defFormat = "yaml"
	}
	if cfg.defFormat != "yaml" && cfg.defFormat != "json" {
		return errors.Errorf("Invalid \"def-format\" specified: %s", cfg.defFormat)
	}

	return errors.New("NotImplemented")
}

func initCodeOnly(ctx context.Context, cfg config) error {
	client := cfg.client

	// Require slug for now. If `dev` is specified and `slug` is not, we should have a new-task prompt.
	if cfg.slug == "" {
		return errors.New("Required flag(s) \"slug\" not set")
	}

	task, err := client.GetTask(ctx, cfg.slug)
	if err != nil {
		return err
	}

	if cfg.file == "" {
		cfg.file, err = promptForNewFileName(task)
		if err != nil {
			return err
		}
	}

	r, err := runtime.Lookup(cfg.file, task.Kind)
	if err != nil {
		return errors.Wrapf(err, "unable to init %q - check that your CLI is up to date", cfg.file)
	}

	if fsx.Exists(cfg.file) {
		if slug, ok := runtime.Slug(cfg.file); ok && slug == task.Slug {
			logger.Step("%s is already linked to %s", cfg.file, cfg.slug)
			suggestNextSteps(cfg.file)
			return nil
		}

		patch, err := patch(cfg.slug, cfg.file)
		if err != nil {
			return err
		}

		if !patch {
			logger.Log("You canceled linking %s to %s", cfg.file, cfg.slug)
			return nil
		}

		buf, err := ioutil.ReadFile(cfg.file)
		if err != nil {
			return err
		}
		code := prependComment(buf, runtime.Comment(r, task))
		// Note: 0644 is ignored because file already exists. Uses a reasonable default just in case.
		if err := ioutil.WriteFile(cfg.file, code, 0644); err != nil {
			return err
		}
		logger.Step("Linked %s to %s", cfg.file, cfg.slug)

		suggestNextSteps(cfg.file)
		return nil
	}

	code, fileMode, err := r.Generate(task)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(cfg.file), 0755); err != nil {
		return err
	}

	if err := ioutil.WriteFile(cfg.file, code, fileMode); err != nil {
		return err
	}

	logger.Step("Created %s", cfg.file)
	suggestNextSteps(cfg.file)
	return nil
}

// prependComment handles writing the linking comment to source code, accounting for shebangs
// (which have to appear first in the file).
func prependComment(source []byte, comment string) []byte {
	var buf bytes.Buffer

	// Regardless of task type, look for a shebang and put comment after it if detected.
	hasShebang := len(source) >= 2 && source[0] == '#' && source[1] == '!'
	appendAfterFirstNewline := hasShebang

	appendComment := func() {
		buf.WriteString(comment)
		buf.WriteRune('\n')
		buf.WriteRune('\n')
	}

	prepended := false
	if !appendAfterFirstNewline {
		appendComment()
		prepended = true
	}
	for _, char := range string(source) {
		buf.WriteRune(char)
		if char == '\n' && appendAfterFirstNewline && !prepended {
			appendComment()
			prepended = true
		}
	}
	return buf.Bytes()
}

func suggestNextSteps(file string) {
	logger.Suggest(
		"⚡ To execute the task locally:",
		"airplane dev %s",
		file,
	)
	logger.Suggest(
		"🛫 To deploy your task to Airplane:",
		"airplane deploy %s",
		file,
	)
}

// Patch asks the user if he would like to patch a file
// and add the airplane special comment.
func patch(slug, file string) (ok bool, err error) {
	err = survey.AskOne(
		&survey.Confirm{
			Message: fmt.Sprintf("Would you like to link %s to %s?", file, slug),
			Help:    "Linking this file will add a special airplane comment.",
			Default: true,
		},
		&ok,
	)
	return
}

func promptForNewFileName(task api.Task) (string, error) {
	fileName := task.Slug + runtime.SuggestExt(task.Kind)

	if cwdIsHome, err := cwdIsHome(); err != nil {
		return "", err
	} else if cwdIsHome {
		// Suggest a subdirectory to avoid putting a file directly into home directory.
		fileName = filepath.Join("airplane", fileName)
	}

	if err := survey.AskOne(
		&survey.Input{
			Message: "Where should the script be created?",
			Default: fileName,
		},
		&fileName,
	); err != nil {
		return "", err
	}
	return fileName, nil
}

func cwdIsHome() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	return cwd == home, nil
}

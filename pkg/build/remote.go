package build

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/airplanedev/archiver"
	"github.com/airplanedev/cli/pkg/api"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/taskdir/definitions"
	"github.com/airplanedev/cli/pkg/utils"
	libBuild "github.com/airplanedev/lib/pkg/build"
	"github.com/airplanedev/lib/pkg/build/ignore"
	"github.com/dustin/go-humanize"
	"github.com/pkg/errors"
	"golang.org/x/sync/singleflight"
)

type contextKey string

const (
	taskSlugContextKey contextKey = "taskSlug"
)

type Deployer struct {
	getRegistryTokenMutex sync.Mutex
	cachedRegistryToken   *api.RegistryTokenResponse

	uploadArchiveSingleFlightGroup singleflight.Group
	uploadedArchives               map[string]string
}

func NewDeployer() *Deployer {
	return &Deployer{
		uploadedArchives: make(map[string]string),
	}
}

func (d *Deployer) remote(ctx context.Context, req Request) (*libBuild.Response, error) {
	ctx = context.WithValue(ctx, taskSlugContextKey, req.Def.Slug)
	if err := confirmBuildRoot(req.Root); err != nil {
		return nil, err
	}
	loader := logger.NewLoader()
	defer loader.Stop()
	loader.Start()

	// Before performing a remote build, we must first update kind/kindOptions
	// since the remote build relies on pulling those from the tasks table (for now).
	if err := updateKindAndOptions(ctx, req.Client, req.Def, req.Shim); err != nil {
		return nil, err
	}

	buildLog(ctx, api.LogLevelInfo, loader, logger.Gray("Authenticating with Airplane..."))
	registry, err := d.getRegistryToken(ctx, req.Client)
	if err != nil {
		return nil, err
	}

	tmpdir, err := ioutil.TempDir("", "airplane-builds-")
	if err != nil {
		return nil, errors.Wrap(err, "creating temporary directory for remote build")
	}
	defer os.RemoveAll(tmpdir)

	archivePath := path.Join(tmpdir, "archive.tar.gz")
	buildLog(ctx, api.LogLevelInfo, loader, logger.Gray("Packaging and uploading %s to build the task...", req.Root))
	if err := archiveTaskDir(req.Def, req.Root, archivePath); err != nil {
		return nil, err
	}

	uploadIDRes, err, _ := d.uploadArchiveSingleFlightGroup.Do(req.Root, func() (interface{}, error) {
		return d.uploadArchive(ctx, req.Client, archivePath, req.Root, loader)
	})

	if err != nil {
		return nil, err
	}
	uploadID := uploadIDRes.(string)

	build, err := req.Client.CreateBuild(ctx, api.CreateBuildRequest{
		TaskID:         req.TaskID,
		SourceUploadID: uploadID,
		Env:            req.TaskEnv,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating build")
	}
	logger.Debug("Created build with id=%s", build.Build.ID)

	if err := waitForBuild(ctx, req.Client, build.Build.ID); err != nil {
		return nil, err
	}

	imageURL := fmt.Sprintf("%s/task-%s:%s",
		registry.Repo,
		libBuild.SanitizeTaskID(req.TaskID),
		build.Build.ID,
	)

	return &libBuild.Response{
		ImageURL: imageURL,
		BuildID:  build.Build.ID,
	}, nil
}

func (d *Deployer) getRegistryToken(ctx context.Context, client *api.Client) (registryToken api.RegistryTokenResponse, err error) {
	d.getRegistryTokenMutex.Lock()
	defer d.getRegistryTokenMutex.Unlock()
	if d.cachedRegistryToken != nil {
		registryToken = *d.cachedRegistryToken
	} else {
		registryToken, err = client.GetRegistryToken(ctx)
		if err != nil {
			return registryToken, errors.Wrap(err, "getting registry token")
		}
		d.cachedRegistryToken = &registryToken
	}
	return registryToken, nil
}

func updateKindAndOptions(ctx context.Context, client *api.Client, def definitions.Definition, shim bool) error {
	task, err := client.GetTask(ctx, def.Slug)
	if err != nil {
		return err
	}

	kind, kindOptions, err := def.GetKindAndOptions()
	if err != nil {
		return err
	}

	// Conditionally instruct the remote builder API to perform a shim-based build.
	if shim {
		kindOptions["shim"] = "true"
	}

	// Normalize entrypoint to `/` regardless of OS.
	// CLI might be run from Windows or not Windows, but remote API is on Linux.
	if ep, ok := kindOptions["entrypoint"].(string); ok {
		kindOptions["entrypoint"] = filepath.ToSlash(ep)
	}

	_, err = client.UpdateTask(ctx, api.UpdateTaskRequest{
		Kind:        kind,
		KindOptions: kindOptions,

		// The following fields are not updated until after the build finishes.
		Slug:                       task.Slug,
		Name:                       task.Name,
		Description:                task.Description,
		Image:                      task.Image,
		Command:                    task.Command,
		Arguments:                  task.Arguments,
		Parameters:                 task.Parameters,
		Constraints:                task.Constraints,
		Env:                        task.Env,
		ResourceRequests:           task.ResourceRequests,
		Resources:                  task.Resources,
		Repo:                       task.Repo,
		RequireExplicitPermissions: task.RequireExplicitPermissions,
		Permissions:                task.Permissions,
		Timeout:                    task.Timeout,
	})
	if err != nil {
		return errors.Wrapf(err, "updating task %s", def.Slug)
	}

	return nil
}

func archiveTaskDir(def definitions.Definition, root string, archivePath string) error {
	// mholt/archiver takes a list of "sources" (files/directories) that will
	// be included in the root of the archive. In our case, we want the root of
	// the archive to be the contents of the task directory, rather than the
	// task directory itself.
	var sources []string
	if files, err := ioutil.ReadDir(root); err != nil {
		return errors.Wrap(err, "inspecting files in task root")
	} else {
		for _, f := range files {
			sources = append(sources, path.Join(root, f.Name()))
		}
	}

	var err error
	arch := archiver.NewTarGz()
	arch.Tar.IncludeFunc, err = ignore.Func(root)
	if err != nil {
		return err
	}

	if err := arch.Archive(sources, archivePath); err != nil {
		return errors.Wrap(err, "building archive")
	}

	return nil
}

func (d *Deployer) uploadArchive(ctx context.Context, client *api.Client, archivePath, rootPath string, loader *logger.Loader) (string, error) {
	// Check if anyone has uploaded an archive for this path.
	uid, ok := d.uploadedArchives[rootPath]
	if ok {
		// Somebody has already uploaded the path. Re-use the upload ID.
		return uid, nil
	}

	loader.Start()

	archive, err := os.OpenFile(archivePath, os.O_RDONLY, 0)
	if err != nil {
		return "", errors.Wrap(err, "opening archive file")
	}
	defer archive.Close()

	info, err := archive.Stat()
	if err != nil {
		return "", errors.Wrap(err, "stat on archive file")
	}
	sizeBytes := int(info.Size())

	buildLog(ctx, api.LogLevelInfo, loader, logger.Gray("Uploading %s build archive...",
		humanize.Bytes(uint64(sizeBytes)),
	))

	upload, err := client.CreateBuildUpload(ctx, api.CreateBuildUploadRequest{
		SizeBytes: sizeBytes,
	})
	if err != nil {
		return "", errors.Wrap(err, "creating upload")
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", upload.WriteOnlyURL, archive)
	if err != nil {
		return "", errors.Wrap(err, "creating GCS upload request")
	}
	req.Header.Add("X-Goog-Content-Length-Range", fmt.Sprintf("0,%d", sizeBytes))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "uploading to GCS")
	}
	defer resp.Body.Close()

	logger.Debug("Upload complete: %s", upload.Upload.URL)
	uploadID := upload.Upload.ID

	// Populate the cache so that we can reuse the upload.
	d.uploadedArchives[rootPath] = uploadID

	return uploadID, nil
}

func waitForBuild(ctx context.Context, client *api.Client, buildID string) error {
	loader := logger.NewLoader()
	defer loader.Stop()
	loader.Start()
	buildLog(ctx, api.LogLevelInfo, loader, logger.Gray("Waiting for builder..."))

	t := time.NewTicker(time.Second)

	var prevToken string
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			r, err := client.GetBuildLogs(ctx, buildID, prevToken)
			if err != nil {
				return errors.Wrap(err, "getting build logs")
			}

			if len(r.Logs) > 0 {
				prevToken = r.PrevPageToken
			}

			api.SortLogs(r.Logs)
			for _, l := range r.Logs {
				text := l.Text
				if strings.HasPrefix(l.Text, "[builder] ") {
					text = logger.Gray(strings.TrimPrefix(text, "[builder] "))
				}

				buildLog(ctx, l.Level, loader, text)
			}

			b, err := client.GetBuild(ctx, buildID)
			if err != nil {
				return errors.Wrap(err, "getting build")
			}

			if b.Build.Status.Stopped() {
				loader.Stop()
				switch b.Build.Status {
				case api.BuildCancelled:
					buildLog(ctx, api.LogLevelInfo, loader, logger.Bold(logger.Yellow("cancelled")))
					return errors.New("Build cancelled")
				case api.BuildFailed:
					buildLog(ctx, api.LogLevelInfo, loader, logger.Bold(logger.Red("failed")))
					return errors.New("Build failed")
				case api.BuildSucceeded:
					buildLog(ctx, api.LogLevelInfo, loader, logger.Bold(logger.Green("succeeded")))
				}

				return nil
			}
			loader.Start()
		}
	}
}

func buildLog(ctx context.Context, level api.LogLevel, loader *logger.Loader, msg string, args ...interface{}) {
	taskSlug := ctx.Value(taskSlugContextKey).(string)
	loaderActive := loader.IsActive()
	loader.Stop()
	buildMsg := fmt.Sprintf("[%s %s] ", logger.Yellow("build"), taskSlug)
	if level == api.LogLevelDebug {
		logger.Log(buildMsg+"["+logger.Blue("debug")+"] "+msg, args...)
	} else {
		logger.Log(buildMsg+msg, args...)
	}
	if loaderActive {
		loader.Start()
	}
}

func confirmBuildRoot(root string) error {
	if home, err := os.UserHomeDir(); err != nil {
		return errors.Wrap(err, "getting home dir")
	} else if home != root {
		return nil
	}
	logger.Warning("This task's root is your home directory — deploying will attempt to upload the entire directory.")
	logger.Warning("Consider moving your task entrypoint to a subdirectory.")
	if ok, err := utils.Confirm("Are you sure?"); err != nil {
		return err
	} else if !ok {
		return errors.New("aborting build")
	}
	return nil
}

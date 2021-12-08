package latest

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/airplanedev/cli/pkg/analytics"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/airplanedev/cli/pkg/version"
)

const releaseURL = "https://api.github.com/repos/airplanedev/cli/releases?per_page=1"

type release struct {
	Name       string `json:"name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// CheckLatest queries the GitHub API for newer releases and prints a warning if the CLI is outdated.
func CheckLatest(ctx context.Context) {
	latest, err := getLatest(ctx)
	if err != nil {
		analytics.ReportError(err)
		logger.Debug("An error ocurred checking for the latest version: %s", err)
		return
	}
	if latest == "" || version.Get() == "<unknown>" {
		// No version found or CLI version unknown - pass silently.
		return
	}
	// Assumes not matching latest means you are behind:
	if latest != "v"+version.Get() {
		logger.Warning("A newer version of the Airplane CLI is available: %s", latest)
		logger.Suggest(
			"Visit the docs for upgrade instructions:",
			"https://docs.airplane.dev/platform/airplane-cli#upgrading-the-cli",
		)
	}
	return
}

func getLatest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", releaseURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	if len(releases) == 0 {
		return "", nil
	}
	for _, release := range releases {
		if release.Draft || release.Prerelease {
			continue
		}
		return release.Name, nil
	}
	return "", nil
}

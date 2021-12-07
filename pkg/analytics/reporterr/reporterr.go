// reporterr is a sub-package to break circular dependency from versions / analytics
package reporterr

import (
	"errors"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/airplanedev/cli/pkg/logger"
	"github.com/getsentry/sentry-go"
)

func ReportError(err error) {
	if ignoreError(err) {
		return
	}
	sentryID := sentry.CaptureException(err)
	if sentryID != nil {
		logger.Debug("Sentry event ID: %s", *sentryID)
	}
}

func ignoreError(err error) bool {
	// For now, all this does is handle survey's interrupt error.
	return errors.Is(err, terminal.InterruptErr)
}

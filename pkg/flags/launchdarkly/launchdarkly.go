package launchdarkly

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/airplanedev/cli/pkg/analytics"
	"github.com/airplanedev/cli/pkg/cli"
	"github.com/airplanedev/cli/pkg/flags"
	"github.com/spf13/viper"

	"github.com/pkg/errors"
	"gopkg.in/launchdarkly/go-sdk-common.v2/lduser"
	"gopkg.in/launchdarkly/go-sdk-common.v2/ldvalue"
	ld "gopkg.in/launchdarkly/go-server-sdk.v5"
)

type Client struct {
	ldc *ld.LDClient
	cfg *cli.Config
}

// Set by Go Releaser.
var (
	launchdarklySDKKey string
)

var _ flags.Flagger = &Client{}

// initialized is 0 if NewClient has not beenP called, otherwise it is 1.
var initialized int32

// NewClient returns a Flagger implementation that is backed by LaunchDarkly.
func NewClient(cfg *cli.Config) (*Client, error) {
	// The underlying LaunchDarkly client expects itself to be used as a singleton
	// since it utilizes a flag cache to reduce the number of API calls.
	//
	// We should only ever create a single instance of this client and dependency
	// inject it. We do not offer a pre-initialized singleton instance because
	// doing so would make it difficult to toggle flags during tests.
	if !atomic.CompareAndSwapInt32(&initialized, 0, 1) {
		return nil, errors.New("unable to initialize LaunchDarkly: NewClient called more than once")
	}

	sdkKey := launchdarklySDKKey
	if sdkKey == "" {
		// Fall back on env var.
		sdkKey = viper.GetString("launchdarkly_sdk_key")
	}
	var ldc *ld.LDClient
	if sdkKey != "" {
		var err error
		ldc, err = ld.MakeClient(sdkKey, 5*time.Second)
		if err != nil {
			return nil, errors.Wrap(err, "unable to instantiate LaunchDarkly client")
		}
	}

	return &Client{
		ldc: ldc,
		cfg: cfg,
	}, nil
}

func (c *Client) Bool(ctx context.Context, flag string, opts ...flags.BoolOpts) bool {
	if c.ldc == nil {
		return false
	}
	o := flags.BoolOpts{}
	if len(opts) > 0 {
		o = opts[0]
	}

	v, err := c.ldc.BoolVariation(flag, c.userFromConfig(ctx), o.Default)
	if err != nil {
		analytics.ReportError(errors.Wrapf(err, "unable to check LaunchDarkly flag %q", flag))
		return o.Default
	}

	return v
}

// userFromConfig generates a user profile that can be used for flag targetting.
func (c *Client) userFromConfig(ctx context.Context) lduser.User {
	info := c.cfg.ParseTokenForAnalytics()

	ldb := lduser.NewUserBuilder(info.UserID).
		Custom("teamID", ldvalue.String(info.TeamID))

	return ldb.Build()
}

func (c *Client) Close() error {
	if c.ldc == nil {
		return nil
	}
	return errors.Wrap(c.ldc.Close(), "unable to close LaunchDarkly client")
}

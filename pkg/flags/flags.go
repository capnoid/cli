package flags

import "context"

// Flaggers are the primary mechanism for dynamically adjusting runtime behavior
// on a per-customer level.
//
// Currently, only boolean feature flags are supported. In the future, we
// may extend this to support multi-variate flags.
//
// Customer metadata, such as the current user's ID or team, can be pulled from
// the context by Flagger implementations. However, whether or not this metadata
// is used during a flag evaluation is implementation-dependent.
//
// For testing, see `NewMock`.
type Flagger interface {
	// Bool returns a boolean representing the state of `flag`. For example:
	//
	// 	if on := flagger.Bool(ctx, "my-flag"); on {
	// 		// ...
	// 	}
	//
	// If an error is encountered, `opts.Default` will be returned.
	Bool(ctx context.Context, flag string, opts ...BoolOpts) bool
}

// BoolOpts provides optional configuration for changing the behavior of
// a Flagger implementation.
type BoolOpts struct {
	// Default determines what to return if an error is encountered while
	// checking a flag.
	//
	// Defaults to `false`.
	Default bool
}

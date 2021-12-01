package flags

import (
	"context"
)

type mock struct {
	returnVal bool
}

var _ Flagger = mock{}

// NewMock returns a mock Flagger that will always return the provided boolean.
//
// The value of `opts.Default` is ignored, since this Flagger will never error.
func NewMock(returnVal bool) Flagger {
	return mock{returnVal: returnVal}
}

func (f mock) Bool(ctx context.Context, flag string, opts ...BoolOpts) bool {
	return f.returnVal
}

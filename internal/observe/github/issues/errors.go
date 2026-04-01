package issues

import "errors"

// ErrFatalObservation signals that observation must abort with a non-zero exit (e.g. exactly
// one --issue and GitHub returns 404). The observe engine propagates this from the GitHub provider.
var ErrFatalObservation = errors.New("fatal observation")

package common

import (
	"context"
	"time"
)

// RetryFunc retry invoking function fn until function returned with no retry request or exhausted attempts
// e.g. when function returns with 'false, nil' -> no retry request, we're done
//      when function returns with 'true, x' -> retry requested, try again
func RetryFunc(ctx context.Context,
	attempts int,
	retryInterval time.Duration,
	fn func(attempt int) (bool, error)) error {

	var err error
	var retry bool

	for attempt := 1; attempt <= attempts; attempt++ {
		retry, err = fn(attempt)

		// if there's no need to retry - we're done
		if !retry {
			return err
		}

		// are we out of time?
		// or context error detected during retries
		if ctx.Err() != nil {

			// return the error if one was provided
			if err != nil {
				return err
			}

			return ctx.Err()
		}

		// wait for another retry
		time.Sleep(retryInterval)
	}

	// attempts exhausted, and we're unsuccessful
	return err
}

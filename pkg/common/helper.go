/*
Copyright 2018 Iguazio Systems Ltd.

Licensed under the Apache License, Version 2.0 (the "License") with
an addition restriction as set forth herein. You may not use this
file except in compliance with the License. You may obtain a copy of
the License at http://www.apache.org/licenses/LICENSE-2.0.

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
implied. See the License for the specific language governing
permissions and limitations under the License.

In addition, you may not use the software for any purposes that are
illegal under applicable law, and the grant of the foregoing license
under the Apache 2.0 license is conditioned upon your compliance with
such restriction.
*/
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

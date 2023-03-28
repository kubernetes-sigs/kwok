/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package wait

import (
	"context"
	"time"

	apimachinerywait "k8s.io/apimachinery/pkg/util/wait"
)

const (
	defaultPollTimeout         = 1 * time.Minute
	defaultPollInterval        = 1 * time.Second
	defaultPollContinueOnError = 5
)

// Options is used to configure the wait
type Options struct {
	// Interval is used to specify the interval for the wait
	Interval time.Duration
	// Timeout is used to specify the timeout for the wait
	Timeout time.Duration
	// Immediate is used to indicate if the wait should be immediate
	Immediate bool
	// ContinueOnError is used to specify the number of times the wait should continue on error
	ContinueOnError int
}

// OptionFunc is a function that can be used to configure the wait
type OptionFunc func(*Options)

// WithContinueOnError configures the number of times the wait should continue on error
func WithContinueOnError(continueOnError int) OptionFunc {
	return func(options *Options) {
		options.ContinueOnError = continueOnError
	}
}

// WithTimeout configures the timeout for the wait
func WithTimeout(timeout time.Duration) OptionFunc {
	return func(options *Options) {
		options.Timeout = timeout
	}
}

// WithInterval configures the interval for the wait
func WithInterval(interval time.Duration) OptionFunc {
	return func(options *Options) {
		options.Interval = interval
	}
}

// WithImmediate configures the wait to be immediate
func WithImmediate() OptionFunc {
	return func(options *Options) {
		options.Immediate = true
	}
}

// Poll polls a condition until it returns true, an error, or the timeout is reached.
func Poll(ctx context.Context, conditionFunc apimachinerywait.ConditionWithContextFunc, opts ...OptionFunc) error {
	options := &Options{
		Interval:        defaultPollInterval,
		Timeout:         defaultPollTimeout,
		ContinueOnError: defaultPollContinueOnError,
		Immediate:       false,
	}

	for _, fn := range opts {
		fn(options)
	}

	cf := conditionFunc

	if options.ContinueOnError != 0 {
		count := 0
		cf = func(ctx context.Context) (bool, error) {
			done, err := conditionFunc(ctx)
			if err != nil {
				count++
				if count == options.ContinueOnError {
					return false, err
				}
			}
			return done, nil
		}
	}

	if options.Immediate {
		return apimachinerywait.PollImmediateWithContext(ctx, options.Interval, options.Timeout, cf)
	}
	return apimachinerywait.PollWithContext(ctx, options.Interval, options.Timeout, cf)
}

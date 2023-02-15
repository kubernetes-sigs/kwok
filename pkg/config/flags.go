/*
Copyright 2022 The Kubernetes Authors.

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

package config

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

// InitFlags initializes the flags for the configuration.
func InitFlags(ctx context.Context, flags *pflag.FlagSet) (context.Context, error) {
	defaultConfigPath := path.Join(WorkDir, consts.ConfigName)
	config := flags.StringArrayP("config", "c", []string{defaultConfigPath}, "config path")
	_ = flags.Parse(os.Args[1:])

	logger := log.FromContext(ctx)
	objs, err := Load(ctx, *config...)
	if err != nil {
		if len(*config) == 1 && (*config)[0] == defaultConfigPath && errors.Is(err, os.ErrNotExist) {
			logger.Debug("Load config",
				"path", *config,
				"err", err,
			)
			return setupContext(ctx, objs), nil
		}
		return nil, err
	}

	if len(objs) == 0 {
		logger.Debug("Load config",
			"path", *config,
			"err", "empty config",
		)
	} else {
		logger.Debug("Load config",
			"path", *config,
		)
	}

	return setupContext(ctx, objs), nil
}

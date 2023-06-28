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
	"os"

	"github.com/spf13/pflag"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// InitFlags initializes the flags for the configuration.
func InitFlags(ctx context.Context, flags *pflag.FlagSet) (context.Context, error) {
	defaultConfigPath := path.RelFromHome(path.Join(WorkDir, consts.ConfigName))
	config := flags.StringArrayP("config", "c", []string{defaultConfigPath}, "config path")
	_ = flags.Parse(os.Args[1:])

	// Expand the all config paths.
	defaultConfigPath, err := path.Expand(defaultConfigPath)
	if err != nil {
		return nil, err
	}
	configPaths := make([]string, 0, len(*config))
	for _, c := range *config {
		if c == "-" {
			configPaths = append(configPaths, c)
			continue
		}
		configPath, err := path.Expand(c)
		if err != nil {
			return nil, err
		}
		configPaths = append(configPaths, configPath)
	}

	configPaths = loadConfig(configPaths, defaultConfigPath, file.Exists(defaultConfigPath))

	logger := log.FromContext(ctx)
	objs, err := Load(ctx, configPaths...)
	if err != nil {
		return nil, err
	}

	if len(objs) == 0 {
		logger.Debug("Load config",
			"path", configPaths,
			"err", "empty config",
		)
	} else {
		logger.Debug("Load config",
			"path", configPaths,
			"count", len(objs),
			"content", objs,
		)
	}

	return setupContext(ctx, objs), nil
}

// loadConfig loads the config paths.
// ~/.kwok/kwok.yaml will be loaded first if it exists.
func loadConfig(configPaths []string, defaultConfigPath string, existDefaultConfig bool) []string {
	if !slices.Contains(configPaths, defaultConfigPath) {
		if existDefaultConfig {
			// If the defaultConfigPath is not specified and the default config exists, it will be loaded first.
			return append([]string{defaultConfigPath}, configPaths...)
		}
	} else {
		if !existDefaultConfig {
			// If the defaultConfigPath is specified and the default config does not exist, it will be removed.
			return slices.Filter(configPaths, func(s string) bool {
				return s != defaultConfigPath
			})
		}
	}
	return configPaths
}

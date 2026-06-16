/*
Copyright 2026 The Kubernetes Authors.

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

package runtime

import (
	"context"
	"slices"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
)

// Install installs the cluster
func (c *Cluster) Install(ctx context.Context) error {
	return c.MkdirAll(c.Workdir())
}

// Uninstall uninstalls the cluster.
func (c *Cluster) Uninstall(ctx context.Context) error {
	// cleanup workdir
	return c.RemoveAll(c.Workdir())
}

// CheckComponentIssues checks and warns about component enable/disable issues.
func (c *Cluster) CheckComponentIssues(ctx context.Context, components []internalversion.Component) error {
	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

	enabled, disabled, err := c.getEnabledAndDisabledComponents(ctx)
	if err != nil {
		return err
	}

	// Get available component names from configuration
	available := utilsslices.Map(conf.Components, func(c internalversion.Component) string {
		return c.Name
	})

	logger := log.FromContext(ctx)

	// Check for disabled components not in available list
	disabled = utilsslices.Unique(disabled)
	if len(disabled) > 0 {
		unknown := utilsslices.Filter(disabled, func(name string) bool {
			return !slices.Contains(available, name)
		})
		if len(unknown) > 0 {
			logger.Warn("Some disabled components are not known to the runtime",
				"components", unknown,
			)
		}
	}

	// Check for enabled components not in final built components
	enabled = utilsslices.Unique(enabled)
	if len(enabled) > 0 {
		actualNames := utilsslices.Map(components, func(component internalversion.Component) string {
			return component.Name
		})
		unused := utilsslices.Filter(enabled, func(name string) bool {
			return !slices.Contains(actualNames, name)
		})
		if len(unused) > 0 {
			logger.Warn("Some enabled components are not used in the runtime",
				"components", unused,
			)
		}
	}
	return nil
}

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

package components

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/util/sets"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

var (
	// ErrBrokenLinks is returned when there are broken links.
	ErrBrokenLinks = fmt.Errorf("broken links dependency detected")
)

// GroupByLinks groups stages by links.
func GroupByLinks(components []internalversion.Component) ([][]internalversion.Component, error) {
	had := sets.NewString()
	next := slices.Clone(components)
	groups := [][]internalversion.Component{}

	for len(next) != 0 {
		current := next
		next = next[:0]
		group := []internalversion.Component{}

		for _, component := range current {
			if len(component.Links) != 0 && !had.HasAll(component.Links...) {
				next = append(next, component)
				continue
			}
			group = append(group, component)
		}
		if len(group) == 0 {
			if len(next) != 0 {
				next := slices.Map(next, func(component internalversion.Component) string {
					return component.Name
				})
				return nil, fmt.Errorf("%w: %v", ErrBrokenLinks, next)
			}
		} else {
			added := slices.Map(group, func(component internalversion.Component) string {
				return component.Name
			})
			had.Insert(added...)
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func extraArgsToStrings(args []internalversion.ExtraArgs) []string {
	return slices.Map(args, func(arg internalversion.ExtraArgs) string {
		return fmt.Sprintf("--%s=%s", arg.Key, arg.Value)
	})
}

// ForeachComponents starts components.
func ForeachComponents(ctx context.Context, cs []internalversion.Component, reverse, order bool, fun func(ctx context.Context, component internalversion.Component) error) error {
	groups, err := GroupByLinks(cs)
	if err != nil {
		return err
	}
	if reverse {
		groups = slices.Reverse(groups)
	}

	if order {
		for _, group := range groups {
			if len(group) == 1 {
				if err := fun(ctx, group[0]); err != nil {
					return err
				}
			} else {
				g, ctx := errgroup.WithContext(ctx)
				for _, component := range group {
					component := component
					g.Go(func() error {
						return fun(ctx, component)
					})
				}
				if err := g.Wait(); err != nil {
					return err
				}
			}
		}
	} else {
		g, ctx := errgroup.WithContext(ctx)
		for _, group := range groups {
			for _, component := range group {
				component := component
				g.Go(func() error {
					return fun(ctx, component)
				})
			}
		}
		if err := g.Wait(); err != nil {
			return err
		}
	}
	return nil
}

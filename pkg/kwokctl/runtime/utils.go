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

package runtime

import (
	"context"
	"io"
	"os"

	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/log"
)

func PullImages(ctx context.Context, command string, images []string, quiet bool) error {
	var out io.Writer = os.Stderr
	if quiet {
		out = nil
	}

	logger := log.FromContext(ctx)

	for _, image := range images {
		err := utils.Exec(ctx, "", utils.IOStreams{},
			command, "inspect",
			image,
		)
		if err != nil {
			logger.Info("Pull image", "image", image)
			err = utils.Exec(ctx, "", utils.IOStreams{
				Out:    out,
				ErrOut: out,
			}, command, "pull",
				image,
			)
			if err != nil {
				return err
			}
		} else {
			logger.Info("Image already exists", "image", image)
		}
	}
	return nil
}

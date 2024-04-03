/*
Copyright 2024 The Kubernetes Authors.

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

package image

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	containerregistryv1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/wzshiming/httpseek"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/progressbar"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

// Pull pulls an image from a registry.
func Pull(ctx context.Context, cacheDir, src, dest string, quiet bool) error {
	logger := log.FromContext(ctx)
	logger = logger.With(
		"image", src,
	)
	logger.Info("Pull")

	var transport = remote.DefaultTransport
	var retry = 10
	transport = httpseek.NewMustReaderTransport(transport, func(req *http.Request, err error) error {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if retry > 0 {
			retry--
			time.Sleep(time.Second)
			return nil
		}
		return err
	})

	if !quiet {
		transport = progressbar.NewTransport(transport)
	}

	o := crane.GetOptions(
		crane.WithContext(ctx),
		crane.WithUserAgent(version.DefaultUserAgent()),
		crane.WithTransport(transport),
		crane.WithPlatform(&containerregistryv1.Platform{
			OS:           "linux",
			Architecture: runtime.GOARCH,
		}),
	)

	ref, err := name.ParseReference(src, o.Name...)
	if err != nil {
		return fmt.Errorf("parsing reference %q: %w", src, err)
	}

	rmt, err := remote.Get(ref, o.Remote...)
	if err != nil {
		return err
	}

	img, err := rmt.Image()
	if err != nil {
		return err
	}
	if cacheDir != "" {
		img = cache.Image(img, cache.NewFilesystemCache(cacheDir))
	}

	err = crane.Save(img, src, dest)
	if err != nil {
		return fmt.Errorf("saving tarball %s: %w", dest, err)
	}

	return nil
}

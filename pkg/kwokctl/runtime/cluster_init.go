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
	"bytes"
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/kwok/kustomize/crd"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// InitCRDs initializes the CRDs.
func (c *Cluster) InitCRDs(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}
	conf := &config.Options

	crds := conf.EnableCRDs
	if len(crds) == 0 {
		return nil
	}

	if c.IsDryRun() {
		dryrun.PrintMessagef("# Init CRDs %s", strings.Join(crds, ","))
		return nil
	}

	clientset, err := c.GetClientset(ctx)
	if err != nil {
		return err
	}

	buf := bytes.NewBuffer(nil)
	for _, name := range crds {
		c, ok := crdDefines[name]
		if !ok {
			return fmt.Errorf("no crd define found for %s", name)
		}
		_, _ = buf.WriteString("\n---\n")
		_, _ = buf.Write(c)
	}

	logger := log.FromContext(ctx)
	logger = logger.With(
		"crds", crds,
	)
	ctx = log.NewContext(ctx, logger)

	loader, err := snapshot.NewLoader(snapshot.LoadConfig{
		Clientset: clientset,
		NoFilers:  true,
	})
	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(buf)
	err = loader.Load(ctx, decoder)
	if err != nil {
		return err
	}

	return nil
}

var crdDefines = map[string][]byte{
	v1alpha1.StageKind:                crd.Stage,
	v1alpha1.AttachKind:               crd.Attach,
	v1alpha1.ClusterAttachKind:        crd.ClusterAttach,
	v1alpha1.ExecKind:                 crd.Exec,
	v1alpha1.ClusterExecKind:          crd.ClusterExec,
	v1alpha1.PortForwardKind:          crd.PortForward,
	v1alpha1.ClusterPortForwardKind:   crd.ClusterPortForward,
	v1alpha1.LogsKind:                 crd.Logs,
	v1alpha1.ClusterLogsKind:          crd.ClusterLogs,
	v1alpha1.ResourceUsageKind:        crd.ResourceUsage,
	v1alpha1.ClusterResourceUsageKind: crd.ClusterResourceUsage,
	v1alpha1.MetricKind:               crd.Metric,
}

// InitCRs initializes the CRs.
func (c *Cluster) InitCRs(ctx context.Context) error {
	config, err := c.Config(ctx)
	if err != nil {
		return err
	}

	if c.IsDryRun() {
		for _, component := range config.Components {
			if component.ManifestContents == nil {
				continue
			}

			dryrun.PrintMessagef("# Set up manifest for %s", component.Name)
		}
		return nil
	}

	var buf strings.Builder
	for _, component := range config.Components {
		if component.ManifestContents == nil {
			continue
		}

		for _, v := range component.ManifestContents {
			_, _ = buf.WriteString("---\n")
			_, _ = buf.WriteString(v)
			_, _ = buf.WriteString("\n")
		}
	}

	if buf.Len() == 0 {
		return nil
	}

	clientset, err := c.GetClientset(ctx)
	if err != nil {
		return err
	}

	loader, err := snapshot.NewLoader(snapshot.LoadConfig{
		Clientset: clientset,
		NoFilers:  true,
	})
	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(strings.NewReader(buf.String()))

	return loader.Load(ctx, decoder)
}

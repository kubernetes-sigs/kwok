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

package kind

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

// WaitReady waits for the cluster to be ready.
func (c *Cluster) WaitReady(ctx context.Context, timeout time.Duration) error {
	if c.IsDryRun() {
		return nil
	}

	var (
		err     error
		waitErr error
		ready   bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, err = c.Ready(ctx)
		if err != nil {
			logger.Debug("Cluster is not ready",
				"err", err,
			)
		}
		return ready, nil
	},
		wait.WithTimeout(timeout),
		wait.WithContinueOnError(10),
		wait.WithInterval(time.Second/2),
	)
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

// Ready returns true if the cluster is ready
func (c *Cluster) Ready(ctx context.Context) (bool, error) {
	ok, err := c.Cluster.Ready(ctx)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	config, err := c.Config(ctx)
	if err != nil {
		return false, err
	}

	// TODO: Only the necessary components are checked for readiness.
	for _, component := range config.Components {
		s, _ := c.InspectComponent(ctx, component.Name)
		if s != runtime.ComponentStatusReady {
			return false, nil
		}
	}
	return true, nil
}

// InspectComponent returns the status of the component
func (c *Cluster) InspectComponent(ctx context.Context, name string) (runtime.ComponentStatus, error) {
	ready, running, _, err := c.inspectComponent(ctx, name)
	if err != nil {
		return runtime.ComponentStatusUnknown, err
	}
	if !running {
		return runtime.ComponentStatusStopped, nil
	}
	if !ready {
		return runtime.ComponentStatusRunning, nil
	}
	return runtime.ComponentStatusReady, nil
}

func (c *Cluster) inspectComponent(ctx context.Context, name string) (ready bool, running bool, exist bool, err error) {
	clientset, err := c.GetClientset(ctx)
	if err != nil {
		return false, false, false, err
	}

	restConfig, err := clientset.ToRESTConfig()
	if err != nil {
		return false, false, false, err
	}

	typedClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return false, false, false, err
	}

	pod, err := typedClient.CoreV1().
		Pods(metav1.NamespaceSystem).
		Get(ctx, c.getComponentName(name), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, false, false, nil
		}
		return false, false, false, err
	}

	if pod.Status.Phase != corev1.PodRunning {
		return false, false, true, nil
	}
	if pod.Status.ContainerStatuses == nil {
		return false, true, true, nil
	}
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false, true, true, nil
		}
	}

	return true, true, true, nil
}

// waitComponentReady waits for a component to be ready
func (c *Cluster) waitComponentReady(ctx context.Context, name string, wantReady bool, timeout time.Duration) error {
	var (
		err     error
		waitErr error
		ready   bool
		running bool
	)
	logger := log.FromContext(ctx)
	waitErr = wait.Poll(ctx, func(ctx context.Context) (bool, error) {
		ready, running, _, err = c.inspectComponent(ctx, name)
		if err != nil {
			logger.Debug("check component ready",
				"component", name,
				"err", err,
			)
			//nolint:nilerr
			return false, nil
		}
		if wantReady {
			return ready, nil
		}
		return !running, nil
	},
		wait.WithTimeout(timeout),
		wait.WithImmediate(),
	)
	if err != nil {
		return err
	}
	if waitErr != nil {
		return waitErr
	}
	return nil
}

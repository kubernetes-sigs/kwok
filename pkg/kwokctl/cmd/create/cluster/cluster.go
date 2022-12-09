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

package cluster

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name    string
	Timeout time.Duration

	internalversion.KwokctlConfigurationOptions
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	flags.KwokctlConfigurationOptions = config.GetKwokctlConfiguration(ctx).Options

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a cluster",
		Long:  "Creates a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().Uint32Var(&flags.KubeApiserverPort, "kube-apiserver-port", flags.KubeApiserverPort, `Port of the apiserver (default random)`)
	cmd.Flags().Uint32Var(&flags.PrometheusPort, "prometheus-port", flags.PrometheusPort, `Port to expose Prometheus metrics`)
	cmd.Flags().BoolVar(&flags.SecurePort, "secure-port", flags.SecurePort, `The apiserver port on which to serve HTTPS with authentication and authorization`)
	cmd.Flags().BoolVar(&flags.QuietPull, "quiet-pull", flags.QuietPull, `Pull without printing progress information`)
	cmd.Flags().BoolVar(&flags.DisableKubeScheduler, "disable-kube-scheduler", flags.DisableKubeScheduler, `Disable the kube-scheduler`)
	cmd.Flags().BoolVar(&flags.DisableKubeControllerManager, "disable-kube-controller-manager", flags.DisableKubeControllerManager, `Disable the kube-controller-manager`)
	cmd.Flags().StringVar(&flags.EtcdImage, "etcd-image", flags.EtcdImage, `Image of etcd, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/etcd:${KWOK_ETCD_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeApiserverImage, "kube-apiserver-image", flags.KubeApiserverImage, `Image of kube-apiserver, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-apiserver:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeControllerManagerImage, "kube-controller-manager-image", flags.KubeControllerManagerImage, `Image of kube-controller-manager, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-controller-manager:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeSchedulerImage, "kube-scheduler-image", flags.KubeSchedulerImage, `Image of kube-scheduler, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-scheduler:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KwokControllerImage, "kwok-controller-image", flags.KwokControllerImage, `Image of kwok-controller, only for docker/nerdctl/kind runtime
'${KWOK_IMAGE_PREFIX}/kwok:${KWOK_VERSION}'
`)
	cmd.Flags().StringVar(&flags.PrometheusImage, "prometheus-image", flags.PrometheusImage, `Image of Prometheus, only for docker/nerdctl/kind runtime
'${KWOK_PROMETHEUS_IMAGE_PREFIX}/prometheus:${KWOK_PROMETHEUS_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KindNodeImage, "kind-node-image", flags.KindNodeImage, `Image of kind node, only for kind runtime
'${KWOK_KIND_NODE_IMAGE_PREFIX}/node:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeApiserverBinary, "kube-apiserver-binary", flags.KubeApiserverBinary, `Binary of kube-apiserver, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.KubeControllerManagerBinary, "kube-controller-manager-binary", flags.KubeControllerManagerBinary, `Binary of kube-controller-manager, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.KubeSchedulerBinary, "kube-scheduler-binary", flags.KubeSchedulerBinary, `Binary of kube-scheduler, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.KwokControllerBinary, "kwok-controller-binary", flags.KwokControllerBinary, `Binary of kwok-controller, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.EtcdBinary, "etcd-binary", flags.EtcdBinary, `Binary of etcd, only for binary runtime`)
	cmd.Flags().StringVar(&flags.EtcdBinaryTar, "etcd-binary-tar", flags.EtcdBinaryTar, `Tar of etcd, if --etcd-binary is set, this is ignored, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.PrometheusBinary, "prometheus-binary", flags.PrometheusBinary, `Binary of Prometheus, only for binary runtime`)
	cmd.Flags().StringVar(&flags.PrometheusBinaryTar, "prometheus-binary-tar", flags.PrometheusBinaryTar, `Tar of Prometheus, if --prometheus-binary is set, this is ignored, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.DockerComposeBinary, "docker-compose-binary", flags.DockerComposeBinary, `Binary of Docker-compose, only for docker runtime
`)
	cmd.Flags().StringVar(&flags.KubeFeatureGates, "kube-feature-gates", flags.KubeFeatureGates, `A set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes
`)
	cmd.Flags().StringVar(&flags.KubeRuntimeConfig, "kube-runtime-config", flags.KubeRuntimeConfig, `A set of key=value pairs that enable or disable built-in APIs
`)
	cmd.Flags().StringVar(&flags.KubeAuditPolicy, "kube-audit-policy", flags.KubeAuditPolicy, "Path to the file that defines the audit policy configuration")
	cmd.Flags().BoolVar(&flags.KubeAuthorization, "kube-authorization", flags.KubeAuthorization, "Enable authorization on secure port")
	cmd.Flags().StringVar(&flags.Runtime, "runtime", flags.Runtime, fmt.Sprintf("Runtime of the cluster (%s)", strings.Join(runtime.DefaultRegistry.List(), " or ")))
	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 30*time.Second, "Timeout for waiting for the cluster to be ready")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	buildRuntime, ok := runtime.DefaultRegistry.Get(flags.Runtime)
	if !ok {
		return fmt.Errorf("runtime %q not found", flags.Runtime)
	}

	rt, err := buildRuntime(name, workdir)
	if err != nil {
		return err
	}

	_, err = rt.Config(ctx)
	if err == nil {
		logger.Info("Cluster already exists")

		if ready, err := rt.Ready(ctx); err == nil && ready {
			logger.Info("Cluster is already ready")
			return nil
		}

		logger.Info("Cluster is not ready yet, will be restarted")
		err = rt.Install(ctx)
		if err != nil {
			logger.Error("Failed to continue install cluster", err)
			return err
		}

		// Down the cluster for restart
		err = rt.Down(ctx)
		if err != nil {
			logger.Error("Failed to down cluster", err)
		}
	} else {
		logger.Info("Creating cluster")
		err = rt.SetConfig(ctx, &flags.KwokctlConfigurationOptions)
		if err != nil {
			logger.Error("Failed to set config", err)
			err0 := rt.Uninstall(ctx)
			if err0 != nil {
				logger.Error("Failed to clean up cluster", err0)
			} else {
				logger.Info("Cluster is cleaned up")
			}
			return err
		}
		err = rt.Save(ctx)
		if err != nil {
			logger.Error("Failed to save config", err)
			err0 := rt.Uninstall(ctx)
			if err0 != nil {
				logger.Error("Failed to clean up cluster", err0)
			} else {
				logger.Info("Cluster is cleaned up")
			}
			return err
		}

		err = rt.Install(ctx)
		if err != nil {
			logger.Error("Failed to setup config", err)
			err0 := rt.Uninstall(ctx)
			if err0 != nil {
				logger.Error("Failed to uninstall cluster", err0)
			} else {
				logger.Info("Cluster is cleaned up")
			}
			return err
		}
	}

	logger.Info("Starting cluster")
	err = rt.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to start cluster %q: %w", name, err)
	}

	err = rt.WaitReady(ctx, flags.Timeout)
	if err != nil {
		return fmt.Errorf("failed wait for cluster %q be ready: %w", name, err)
	}

	logger.Info("Cluster is ready")

	fmt.Fprintf(os.Stderr, `You can now use your cluster with:

    kubectl config use-context %s

Thanks for using kwok!
`, name)
	return nil
}

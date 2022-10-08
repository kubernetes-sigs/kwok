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
	"strings"
	"time"

	"github.com/spf13/cobra"

	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/kwokctl/utils"
	"sigs.k8s.io/kwok/pkg/kwokctl/vars"
	"sigs.k8s.io/kwok/pkg/logger"
)

type flagpole struct {
	Name                        string
	KubeApiserverPort           uint32
	PrometheusPort              uint32
	SecurePort                  bool
	QuietPull                   bool
	EtcdImage                   string
	KubeApiserverImage          string
	KubeControllerManagerImage  string
	KubeSchedulerImage          string
	KwokControllerImage         string
	PrometheusImage             string
	KindNodeImage               string
	KubeApiserverBinary         string
	KubeControllerManagerBinary string
	KubeSchedulerBinary         string
	KwokControllerBinary        string
	EtcdBinary                  string
	EtcdBinaryTar               string
	PrometheusBinary            string
	PrometheusBinaryTar         string
	Runtime                     string
	KubeFeatureGates            string
	KubeRuntimeConfig           string
	KubeAuditPolicy             string
	Timeout                     time.Duration
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(logger logger.Logger) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a cluster",
		Long:  "Creates a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = vars.DefaultCluster
			return runE(cmd.Context(), logger, flags)
		},
	}
	cmd.Flags().Uint32Var(&flags.KubeApiserverPort, "kube-apiserver-port", uint32(vars.KubeApiserverPort), `Port of the apiserver (default random)`)
	cmd.Flags().Uint32Var(&flags.PrometheusPort, "prometheus-port", uint32(vars.PrometheusPort), `Port to expose Prometheus metrics`)
	cmd.Flags().BoolVar(&flags.SecurePort, "secure-port", vars.SecurePort, `Apiserver use TLS`)
	cmd.Flags().BoolVar(&flags.QuietPull, "quiet-pull", vars.QuietPull, `Pull without printing progress information`)
	cmd.Flags().StringVar(&flags.EtcdImage, "etcd-image", vars.EtcdImage, `Image of etcd, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/etcd:${KWOK_ETCD_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeApiserverImage, "kube-apiserver-image", vars.KubeApiserverImage, `Image of kube-apiserver, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-apiserver:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeControllerManagerImage, "kube-controller-manager-image", vars.KubeControllerManagerImage, `Image of kube-controller-manager, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-controller-manager:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeSchedulerImage, "kube-scheduler-image", vars.KubeSchedulerImage, `Image of kube-scheduler, only for docker/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-scheduler:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KwokControllerImage, "kwok-controller-image", vars.KwokControllerImage, `Image of kwok-controller, only for docker/nerdctl/kind runtime
'${KWOK_IMAGE_PREFIX}/kwok:${KWOK_VERSION}'
`)
	cmd.Flags().StringVar(&flags.PrometheusImage, "prometheus-image", vars.PrometheusImage, `Image of Prometheus, only for docker/nerdctl/kind runtime
'${KWOK_PROMETHEUS_IMAGE_PREFIX}/prometheus:${KWOK_PROMETHEUS_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KindNodeImage, "kind-node-image", vars.KindNodeImage, `Image of kind node, only for kind runtime
'${KWOK_KIND_NODE_IMAGE_PREFIX}/node:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.KubeApiserverBinary, "kube-apiserver-binary", vars.KubeApiserverBinary, `Binary of kube-apiserver, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.KubeControllerManagerBinary, "kube-controller-manager-binary", vars.KubeControllerManagerBinary, `Binary of kube-controller-manager, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.KubeSchedulerBinary, "kube-scheduler-binary", vars.KubeSchedulerBinary, `Binary of kube-scheduler, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.KwokControllerBinary, "kwok-controller-binary", vars.KwokControllerBinary, `Binary of kwok-controller, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.EtcdBinary, "etcd-binary", vars.EtcdBinary, `Binary of etcd, only for binary runtime`)
	cmd.Flags().StringVar(&flags.EtcdBinaryTar, "etcd-binary-tar", vars.EtcdBinaryTar, `Tar of etcd, if --etcd-binary is set, this is ignored, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.PrometheusBinary, "prometheus-binary", vars.PrometheusBinary, `Binary of Prometheus, only for binary runtime`)
	cmd.Flags().StringVar(&flags.PrometheusBinaryTar, "prometheus-binary-tar", vars.PrometheusBinaryTar, `Tar of Prometheus, if --prometheus-binary is set, this is ignored, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.KubeFeatureGates, "kube-feature-gates", vars.KubeFeatureGates, `A set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes
`)
	cmd.Flags().StringVar(&flags.KubeRuntimeConfig, "kube-runtime-config", vars.KubeRuntimeConfig, `A set of key=value pairs that enable or disable built-in APIs
`)
	cmd.Flags().StringVar(&flags.KubeAuditPolicy, "kube-audit-policy", vars.KubeAuditPolicy, "Path to the file that defines the audit policy configuration")
	cmd.Flags().StringVar(&flags.Runtime, "runtime", vars.Runtime, fmt.Sprintf("Runtime of the cluster (%s)", strings.Join(runtime.DefaultRegistry.List(), " or ")))
	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 30*time.Second, "Timeout for waiting for the cluster to be ready")
	return cmd
}

func runE(ctx context.Context, logger logger.Logger, flags *flagpole) error {
	name := fmt.Sprintf("%s-%s", vars.ProjectName, flags.Name)
	workdir := utils.PathJoin(vars.ClustersDir, flags.Name)

	buildRuntime, ok := runtime.DefaultRegistry.Get(flags.Runtime)
	if !ok {
		return fmt.Errorf("runtime %q not found", flags.Runtime)
	}

	rt, err := buildRuntime(name, workdir, logger)
	if err != nil {
		return err
	}

	_, err = rt.Config()
	if err == nil {
		logger.Printf("Cluster %q already exists", name)

		if ready, err := rt.Ready(ctx); err == nil && ready {
			logger.Printf("Cluster %q is already ready", name)
			return nil
		}

		logger.Printf("Cluster %q is not ready yet, will be restarted", name)
		err = rt.Install(ctx)
		if err != nil {
			logger.Printf("Failed to continue install cluster %q: %v", name, err)
			return err
		}

		// Down the cluster for restart
		err = rt.Down(ctx)
		if err != nil {
			logger.Printf("Failed to down cluster %q: %v", name, err)
		}
	} else {
		logger.Printf("Creating cluster %q", name)
		err = rt.Init(ctx, runtime.Config{
			Name:                        name,
			KubeApiserverPort:           flags.KubeApiserverPort,
			Workdir:                     workdir,
			Runtime:                     flags.Runtime,
			PrometheusImage:             flags.PrometheusImage,
			EtcdImage:                   flags.EtcdImage,
			KubeApiserverImage:          flags.KubeApiserverImage,
			KubeControllerManagerImage:  flags.KubeControllerManagerImage,
			KubeSchedulerImage:          flags.KubeSchedulerImage,
			KwokControllerImage:         flags.KwokControllerImage,
			KindNodeImage:               flags.KindNodeImage,
			KubeApiserverBinary:         flags.KubeApiserverBinary,
			KubeControllerManagerBinary: flags.KubeControllerManagerBinary,
			KubeSchedulerBinary:         flags.KubeSchedulerBinary,
			KwokControllerBinary:        flags.KwokControllerBinary,
			EtcdBinary:                  flags.EtcdBinary,
			EtcdBinaryTar:               flags.EtcdBinaryTar,
			PrometheusBinary:            flags.PrometheusBinary,
			PrometheusBinaryTar:         flags.PrometheusBinaryTar,
			CacheDir:                    vars.CacheDir,
			SecretPort:                  flags.SecurePort,
			QuietPull:                   flags.QuietPull,
			PrometheusPort:              flags.PrometheusPort,
			FeatureGates:                flags.KubeFeatureGates,
			RuntimeConfig:               flags.KubeRuntimeConfig,
			AuditPolicy:                 flags.KubeAuditPolicy,
		})
		if err != nil {
			logger.Printf("Failed to setup config %q: %v", name, err)
			err0 := rt.Uninstall(ctx)
			if err0 != nil {
				logger.Printf("Failed to clean up cluster %q: %v", name, err0)
			} else {
				logger.Printf("Cleaned up %q", name)
			}
			return err
		}

		err = rt.Install(ctx)
		if err != nil {
			logger.Printf("Failed to setup config %q: %v", name, err)
			err0 := rt.Uninstall(ctx)
			if err0 != nil {
				logger.Printf("Failed to uninstall cluster %q: %v", name, err0)
			} else {
				logger.Printf("Cleaned up %q", name)
			}
			return err
		}
	}

	logger.Printf("Starting cluster %q", name)
	err = rt.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to start cluster %q: %w", name, err)
	}

	err = rt.WaitReady(ctx, flags.Timeout)
	if err != nil {
		return fmt.Errorf("failed wait for cluster %q be ready: %w", name, err)
	}

	logger.Printf("Cluster %q is ready", name)
	logger.Printf("You can now use your cluster with:")
	logger.Printf("")
	logger.Printf("    kubectl config use-context %s", name)
	logger.Printf("")
	logger.Printf("Thanks for using kwok!")
	return nil
}

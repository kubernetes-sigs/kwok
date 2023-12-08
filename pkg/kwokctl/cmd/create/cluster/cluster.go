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

// Package cluster contains a command to create a cluster.
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
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name       string
	Timeout    time.Duration
	Wait       time.Duration
	Kubeconfig string

	*internalversion.KwokctlConfiguration
}

// NewCommand returns a new cobra.Command for cluster creation
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	flags.KwokctlConfiguration = config.GetKwokctlConfiguration(ctx)
	flags.Kubeconfig = path.RelFromHome(kubeconfig.GetRecommendedKubeconfigPath())

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "cluster",
		Short: "Creates a cluster",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().Uint32Var(&flags.Options.KubeApiserverPort, "kube-apiserver-port", flags.Options.KubeApiserverPort, `Port of the apiserver (default random)`)
	cmd.Flags().Uint32Var(&flags.Options.PrometheusPort, "prometheus-port", flags.Options.PrometheusPort, `Port to expose Prometheus metrics`)
	cmd.Flags().Uint32Var(&flags.Options.JaegerPort, "jaeger-port", flags.Options.JaegerPort, `Port to expose Jaeger UI`)
	cmd.Flags().BoolVar(&flags.Options.SecurePort, "secure-port", flags.Options.SecurePort, `The apiserver port on which to serve HTTPS with authentication and authorization, is not available before Kubernetes 1.13.0`)
	cmd.Flags().BoolVar(&flags.Options.QuietPull, "quiet-pull", flags.Options.QuietPull, `Pull without printing progress information`)
	cmd.Flags().StringVar(&flags.Options.KubeSchedulerConfig, "kube-scheduler-config", flags.Options.KubeSchedulerConfig, `Path to a kube-scheduler configuration file`)
	cmd.Flags().BoolVar(&flags.Options.DisableKubeScheduler, "disable-kube-scheduler", flags.Options.DisableKubeScheduler, `Disable the kube-scheduler`)
	cmd.Flags().BoolVar(&flags.Options.DisableKubeControllerManager, "disable-kube-controller-manager", flags.Options.DisableKubeControllerManager, `Disable the kube-controller-manager`)
	cmd.Flags().StringVar(&flags.Options.EtcdImage, "etcd-image", flags.Options.EtcdImage, `Image of etcd, only for docker/podman/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/etcd:${KWOK_ETCD_VERSION}'
`)
	cmd.Flags().Uint32Var(&flags.Options.EtcdPort, "etcd-port", flags.Options.EtcdPort, `Port of etcd given to the host. The behavior is unstable for kind/kind-podman runtime and may be modified in the future`)
	cmd.Flags().StringVar(&flags.Options.KubeApiserverImage, "kube-apiserver-image", flags.Options.KubeApiserverImage, `Image of kube-apiserver, only for docker/podman/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-apiserver:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().StringVar(&flags.Options.KubeControllerManagerImage, "kube-controller-manager-image", flags.Options.KubeControllerManagerImage, `Image of kube-controller-manager, only for docker/podman/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-controller-manager:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().Uint32Var(&flags.Options.KubeControllerManagerPort, "kube-controller-manager-port", flags.Options.KubeControllerManagerPort, `Port of kube-controller-manager given to the host, only for binary and docker/podman/nerdctl runtime`)
	cmd.Flags().StringVar(&flags.Options.KubeSchedulerImage, "kube-scheduler-image", flags.Options.KubeSchedulerImage, `Image of kube-scheduler, only for docker/podman/nerdctl runtime
'${KWOK_KUBE_IMAGE_PREFIX}/kube-scheduler:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().Uint32Var(&flags.Options.KubeSchedulerPort, "kube-scheduler-port", flags.Options.KubeSchedulerPort, `Port of kube-scheduler given to the host, only for binary and docker/podman/nerdctl runtime`)
	cmd.Flags().StringVar(&flags.Options.KwokControllerImage, "kwok-controller-image", flags.Options.KwokControllerImage, `Image of kwok-controller, only for docker/podman/nerdctl/kind/kind-podman runtime
'${KWOK_IMAGE_PREFIX}/kwok:${KWOK_VERSION}'
`)
	cmd.Flags().StringVar(&flags.Options.PrometheusImage, "prometheus-image", flags.Options.PrometheusImage, `Image of Prometheus, only for docker/podman/nerdctl/kind/kind-podman runtime
'${KWOK_PROMETHEUS_IMAGE_PREFIX}/prometheus:${KWOK_PROMETHEUS_VERSION}'
`)
	cmd.Flags().StringVar(&flags.Options.JaegerImage, "jaeger-image", flags.Options.JaegerImage, `Image of Jaeger, only for docker/podman/nerdctl/kind/kind-podman runtime
'${KWOK_JAEGER_IMAGE_PREFIX}/all-in-one:${KWOK_JAEGER_VERSION}'
`)
	cmd.Flags().Uint32Var(&flags.Options.KwokControllerPort, "controller-port", flags.Options.KwokControllerPort, `Port of kwok-controller given to the host`)
	cmd.Flags().StringVar(&flags.Options.KindNodeImage, "kind-node-image", flags.Options.KindNodeImage, `Image of kind node, only for kind/kind-podman runtime
'${KWOK_KIND_NODE_IMAGE_PREFIX}/node:${KWOK_KUBE_VERSION}'
`)
	cmd.Flags().Uint32Var(&flags.Options.DashboardPort, "dashboard-port", flags.Options.DashboardPort, `Port of dashboard given to the host`)
	cmd.Flags().StringVar(&flags.Options.DashboardImage, "dashboard-image", flags.Options.DashboardImage, `Image of dashboard, only for docker/podman/nerdctl/kind/kind-podman runtime
'${KWOK_DASHBOARD_IMAGE_PREFIX}/dashboard:${KWOK_DASHBOARD_VERSION}'
`)
	cmd.Flags().StringVar(&flags.Options.KubeApiserverBinary, "kube-apiserver-binary", flags.Options.KubeApiserverBinary, `Binary of kube-apiserver, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.Options.KubeControllerManagerBinary, "kube-controller-manager-binary", flags.Options.KubeControllerManagerBinary, `Binary of kube-controller-manager, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.Options.KubeSchedulerBinary, "kube-scheduler-binary", flags.Options.KubeSchedulerBinary, `Binary of kube-scheduler, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.Options.KwokControllerBinary, "kwok-controller-binary", flags.Options.KwokControllerBinary, `Binary of kwok-controller, only for binary runtime
`)
	cmd.Flags().StringVar(&flags.Options.EtcdBinary, "etcd-binary", flags.Options.EtcdBinary, `Binary of etcd, only for binary runtime`)
	cmd.Flags().StringVar(&flags.Options.EtcdBinaryTar, "etcd-binary-tar", flags.Options.EtcdBinaryTar, `Tar of etcd, if --etcd-binary is set, this is ignored, only for binary runtime
`)
	_ = cmd.Flags().MarkDeprecated("etcd-binary-tar", "--etcd-binary-tar will be removed in a future release, please use --etcd-binary instead")
	cmd.Flags().StringVar(&flags.Options.PrometheusBinary, "prometheus-binary", flags.Options.PrometheusBinary, `Binary of Prometheus, only for binary runtime`)
	cmd.Flags().StringVar(&flags.Options.PrometheusBinaryTar, "prometheus-binary-tar", flags.Options.PrometheusBinaryTar, `Tar of Prometheus, if --prometheus-binary is set, this is ignored, only for binary runtime
`)
	_ = cmd.Flags().MarkDeprecated("prometheus-binary-tar", "--prometheus-binary-tar will be removed in a future release, please use --prometheus-binary instead")
	cmd.Flags().StringVar(&flags.Options.JaegerBinary, "jaeger-binary", flags.Options.JaegerBinary, `Binary of Jaeger, only for binary runtime`)
	cmd.Flags().StringVar(&flags.Options.JaegerBinaryTar, "jaeger-binary-tar", flags.Options.JaegerBinaryTar, `Tar of Jaeger, if --jaeger-binary is set, this is ignored, only for binary runtime
`)
	_ = cmd.Flags().MarkDeprecated("jaeger-binary-tar", "--jaeger-binary-tar will be removed in a future release, please use --jaeger-binary instead")
	cmd.Flags().StringVar(&flags.Options.DockerComposeBinary, "docker-compose-binary", flags.Options.DockerComposeBinary, `Binary of Docker-compose, only for docker runtime
`)
	_ = cmd.Flags().MarkDeprecated("docker-compose-binary", "docker compose will be removed in a future release")

	cmd.Flags().StringVar(&flags.Options.KindBinary, "kind-binary", flags.Options.KindBinary, `Binary of kind, only for kind/kind-podman runtime
`)
	cmd.Flags().StringVar(&flags.Options.KubeFeatureGates, "kube-feature-gates", flags.Options.KubeFeatureGates, `A set of key=value pairs that describe feature gates for alpha/experimental features of Kubernetes`)
	cmd.Flags().StringVar(&flags.Options.KubeRuntimeConfig, "kube-runtime-config", flags.Options.KubeRuntimeConfig, `A set of key=value pairs that enable or disable built-in APIs`)
	cmd.Flags().StringVar(&flags.Options.KubeAuditPolicy, "kube-audit-policy", flags.Options.KubeAuditPolicy, "Path to the file that defines the audit policy configuration")
	cmd.Flags().BoolVar(&flags.Options.KubeAuthorization, "kube-authorization", flags.Options.KubeAuthorization, "Enable authorization for kube-apiserver, only for non kind/kind-podman runtime")
	cmd.Flags().BoolVar(&flags.Options.KubeAdmission, "kube-admission", flags.Options.KubeAdmission, "Enable admission for kube-apiserver, only for non kind/kind-podman runtime")
	cmd.Flags().StringVar(&flags.Options.Runtime, "runtime", flags.Options.Runtime, fmt.Sprintf("Runtime of the cluster (%s)", strings.Join(runtime.DefaultRegistry.List(), " or ")))
	cmd.Flags().DurationVar(&flags.Timeout, "timeout", 0, "Timeout for waiting for the cluster to be created")
	cmd.Flags().DurationVar(&flags.Wait, "wait", 0, "Wait for the cluster to be ready")
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", flags.Kubeconfig, "The path to the kubeconfig file will be added to the newly created cluster and set to current-context")
	cmd.Flags().BoolVar(&flags.Options.DisableQPSLimits, "disable-qps-limits", flags.Options.DisableQPSLimits, "Disable QPS limits for components")
	cmd.Flags().StringSliceVar(&flags.Options.EnableCRDs, "enable-crds", flags.Options.EnableCRDs, "List of CRDs to enable")
	cmd.Flags().UintVar(&flags.Options.NodeLeaseDurationSeconds, "node-lease-duration-seconds", flags.Options.NodeLeaseDurationSeconds, "Duration of node lease in seconds")
	cmd.Flags().StringSliceVar(&flags.Options.EnableStageForRefs, "enable-stage-for-refs", flags.Options.EnableStageForRefs, "List of refs to enable stage for")

	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	var err error
	if flags.Kubeconfig != "" {
		flags.Kubeconfig, err = path.Expand(flags.Kubeconfig)
		if err != nil {
			return err
		}
	}

	gctx := ctx
	if flags.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, flags.Timeout)
		defer cancel()
	}

	// Choose runtime
	var rt runtime.Runtime
	if flags.Options.Runtime == "" {
		errs := make([]error, 0, len(flags.Options.Runtimes))
		for _, r := range flags.Options.Runtimes {
			buildRuntime, ok := runtime.DefaultRegistry.Get(r)
			if !ok {
				err = fmt.Errorf("runtime %q not found", flags.Options.Runtime)
				errs = append(errs, err)
				continue
			}
			if rt, err = buildRuntime(name, workdir); err != nil {
				errs = append(errs, err)
				continue
			}
			if err = rt.Available(ctx); err != nil {
				errs = append(errs, err)
				continue
			}
			flags.Options.Runtime = r
			logger.Debug("Detected runtime available", "runtime", flags.Options.Runtime)
			break
		}
		if flags.Options.Runtime == "" {
			return fmt.Errorf("runtime %v not available: %v", flags.Options.Runtimes, errs)
		}
	} else {
		buildRuntime, ok := runtime.DefaultRegistry.Get(flags.Options.Runtime)
		if !ok {
			return fmt.Errorf("runtime %q not found", flags.Options.Runtime)
		}
		rt, err = buildRuntime(name, workdir)
		if err != nil {
			return fmt.Errorf("runtime %v not available: %w", flags.Options.Runtime, err)
		}
	}

	// Set up the cluster
	_, err = rt.Config(ctx)
	exist := err == nil
	cleanUp := func() {}
	if exist {
		logger.Info("Cluster already exists")
		if ready, err := rt.Ready(ctx); err == nil && ready {
			logger.Info("Cluster is already ready")
			return nil
		}
		logger.Info("Cluster is not ready yet, continue it")
	} else {
		cleanUp = func() {
			subCtx := context.Background()
			err := rt.Uninstall(subCtx)
			if err != nil {
				logger.Error("Failed to clean up cluster", err)
			} else {
				logger.Info("Cluster is cleaned up")
			}
		}
		err = rt.SetConfig(ctx, flags.KwokctlConfiguration)
		if err != nil {
			logger.Error("Failed to set config", err)
			cleanUp()
			return err
		}
		err = rt.Save(ctx)
		if err != nil {
			logger.Error("Failed to save config", err)
			cleanUp()
			return err
		}
	}

	// Create the cluster
	start := time.Now()
	logger.Info("Cluster is creating")
	err = rt.Install(ctx)
	if err != nil {
		logger.Error("Failed to setup config", err)
		cleanUp()
		return err
	}
	if flags.Kubeconfig != "" {
		setContext := func() {
			err = rt.AddContext(ctx, flags.Kubeconfig)
			if err != nil {
				logger.Error("Failed to add context to kubeconfig", err,
					"kubeconfig", flags.Kubeconfig,
				)
			} else {
				logger.Debug("Added context to kubeconfig",
					"kubeconfig", flags.Kubeconfig,
				)
			}
		}

		if flags.Options.Runtime == consts.RuntimeTypeKind ||
			flags.Options.Runtime == consts.RuntimeTypeKindPodman {
			// override kubeconfig for kind
			defer setContext()
		} else {
			setContext()
		}
	}
	logger.Info("Cluster is created",
		"elapsed", time.Since(start),
	)

	// Start the cluster
	start = time.Now()
	logger.Info("Cluster is starting")
	err = rt.Up(ctx)
	if err != nil {
		return fmt.Errorf("failed to start cluster %q: %w", name, err)
	}
	logger.Info("Cluster is started",
		"elapsed", time.Since(start),
	)

	err = rt.InitCRDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to init crds %q: %w", name, err)
	}

	// Wait for cluster to be ready
	if flags.Wait > 0 {
		start = time.Now()
		logger.Info("Waiting for cluster to be ready")
		err = rt.WaitReady(gctx, flags.Wait)
		if err != nil {
			logger.Error("Failed to wait for cluster to be ready", err,
				"elapsed", time.Since(start),
			)
		} else {
			logger.Info("Cluster is ready",
				"elapsed", time.Since(start),
			)
		}
	}

	if log.IsTerminal() && flags.Kubeconfig != "" && !rt.IsDryRun() {
		_, _ = fmt.Fprintf(os.Stderr, `You can now use your cluster with:

	kubectl cluster-info --context %s

Thanks for using kwok!
`, name)
	}
	return nil
}

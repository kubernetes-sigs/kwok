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

// Package cmd defines a root command for the kwok.
package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/clock"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/kwok/server"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
	"sigs.k8s.io/kwok/pkg/utils/wait"
	"sigs.k8s.io/kwok/stages"
)

type flagpole struct {
	Kubeconfig string
	Master     string

	*internalversion.KwokConfiguration
}

// NewCommand returns a new cobra.Command for root
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	flags.KwokConfiguration = config.GetKwokConfiguration(ctx)

	cmd := &cobra.Command{
		Args:          cobra.NoArgs,
		Use:           "kwok",
		Short:         "kwok is a tool for simulating the lifecycle of fake nodes, pods, and other Kubernetes API resources.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.DisplayVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), flags)
		},
	}

	flags.Kubeconfig = path.RelFromHome(kubeconfig.GetRecommendedKubeconfigPath())

	cmd.Flags().StringVar(&flags.Options.CIDR, "cidr", flags.Options.CIDR, "CIDR of the pod ip")
	cmd.Flags().StringVar(&flags.Options.NodeIP, "node-ip", flags.Options.NodeIP, "IP of the node")
	cmd.Flags().StringVar(&flags.Options.NodeName, "node-name", flags.Options.NodeName, "Name of the node")
	cmd.Flags().IntVar(&flags.Options.NodePort, "node-port", flags.Options.NodePort, "Port of the node")
	cmd.Flags().StringVar(&flags.Options.TLSCertFile, "tls-cert-file", flags.Options.TLSCertFile, "File containing the default x509 Certificate for HTTPS")
	cmd.Flags().StringVar(&flags.Options.TLSPrivateKeyFile, "tls-private-key-file", flags.Options.TLSPrivateKeyFile, "File containing the default x509 private key matching --tls-cert-file")
	cmd.Flags().BoolVar(&flags.Options.ManageAllNodes, "manage-all-nodes", flags.Options.ManageAllNodes, "All nodes will be watched and managed. It's conflicted with manage-nodes-with-annotation-selector and manage-nodes-with-label-selector.")
	cmd.Flags().StringVar(&flags.Options.ManageNodesWithAnnotationSelector, "manage-nodes-with-annotation-selector", flags.Options.ManageNodesWithAnnotationSelector, "Nodes that match the annotation selector will be watched and managed. It's conflicted with manage-all-nodes.")
	cmd.Flags().StringVar(&flags.Options.ManageNodesWithLabelSelector, "manage-nodes-with-label-selector", flags.Options.ManageNodesWithLabelSelector, "Nodes that match the label selector will be watched and managed. It's conflicted with manage-all-nodes.")
	cmd.Flags().StringVar(&flags.Options.DisregardStatusWithAnnotationSelector, "disregard-status-with-annotation-selector", flags.Options.DisregardStatusWithAnnotationSelector, "All node/pod status excluding the ones that match the annotation selector will be watched and managed.")
	cmd.Flags().StringVar(&flags.Options.DisregardStatusWithLabelSelector, "disregard-status-with-label-selector", flags.Options.DisregardStatusWithLabelSelector, "All node/pod status excluding the ones that match the label selector will be watched and managed.")
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", flags.Kubeconfig, "Path to the kubeconfig file to use")
	cmd.Flags().StringVar(&flags.Master, "master", flags.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	cmd.Flags().StringVar(&flags.Options.ServerAddress, "server-address", flags.Options.ServerAddress, "Address to expose the server on")
	cmd.Flags().UintVar(&flags.Options.NodeLeaseDurationSeconds, "node-lease-duration-seconds", flags.Options.NodeLeaseDurationSeconds, "Duration of node lease seconds")

	cmd.Flags().BoolVar(&flags.Options.EnableCNI, "experimental-enable-cni", flags.Options.EnableCNI, "Experimental support for getting pod ip from CNI, for CNI-related components, Only works with Linux")
	if config.GOOS != "linux" {
		_ = cmd.Flags().MarkHidden("experimental-enable-cni")
	}
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	logger := log.FromContext(ctx)

	if flags.Kubeconfig != "" {
		var err error
		flags.Kubeconfig, err = path.Expand(flags.Kubeconfig)
		if err != nil {
			return err
		}
		f, err := os.Stat(flags.Kubeconfig)
		if err != nil || f.IsDir() {
			logger.Warn("Failed to get kubeconfig file or it is a directory", "kubeconfig", flags.Kubeconfig)
			flags.Kubeconfig = ""
		}
	}

	if flags.Kubeconfig == "" && flags.Master == "" {
		logger.Warn("Neither --kubeconfig nor --master was specified")
		logger.Info("Using the inClusterConfig")
	}
	clientset, err := client.NewClientset(flags.Master, flags.Kubeconfig)
	if err != nil {
		return err
	}

	typedClient, err := clientset.ToTypedClient()
	if err != nil {
		return err
	}
	typedKwokClient, err := clientset.ToTypedKwokClient()
	if err != nil {
		return err
	}

	if flags.Options.ManageAllNodes {
		if flags.Options.ManageNodesWithAnnotationSelector != "" || flags.Options.ManageNodesWithLabelSelector != "" {
			logger.Error("manage-all-nodes is conflicted with manage-nodes-with-annotation-selector and manage-nodes-with-label-selector.", nil)
			os.Exit(1)
		}
		logger.Info("Watch all nodes")
	} else if flags.Options.ManageNodesWithAnnotationSelector != "" || flags.Options.ManageNodesWithLabelSelector != "" {
		logger.Info("Watch nodes",
			"annotation", flags.Options.ManageNodesWithAnnotationSelector,
			"label", flags.Options.ManageNodesWithLabelSelector,
		)
	}

	err = waitForReady(ctx, typedClient)
	if err != nil {
		return err
	}

	stagesData := config.FilterWithTypeFromContext[*internalversion.Stage](ctx)

	nodeStages := filterStages(stagesData, "v1", "Node")
	if len(nodeStages) == 0 {
		nodeStages, err = controllers.NewStagesFromYaml([]byte(stages.DefaultNodeStages))
		if err != nil {
			return err
		}
		if flags.Options.NodeLeaseDurationSeconds == 0 {
			nodeHeartbeatStages, err := controllers.NewStagesFromYaml([]byte(stages.DefaultNodeHeartbeatStages))
			if err != nil {
				return err
			}
			nodeStages = append(nodeStages, nodeHeartbeatStages...)
		}
	}
	podStages := filterStages(stagesData, "v1", "Pod")
	if len(podStages) == 0 {
		podStages, err = controllers.NewStagesFromYaml([]byte(stages.DefaultPodStages))
		if err != nil {
			return err
		}
	}

	id, err := controllers.Identity()
	if err != nil {
		return err
	}
	ctx = log.NewContext(ctx, logger.With("id", id))

	metrics := config.FilterWithTypeFromContext[*internalversion.Metric](ctx)

	ctr, err := controllers.NewController(controllers.Config{
		Clock:                                 clock.RealClock{},
		TypedClient:                           typedClient,
		EnableCNI:                             flags.Options.EnableCNI,
		ManageAllNodes:                        flags.Options.ManageAllNodes,
		ManageNodesWithAnnotationSelector:     flags.Options.ManageNodesWithAnnotationSelector,
		ManageNodesWithLabelSelector:          flags.Options.ManageNodesWithLabelSelector,
		DisregardStatusWithAnnotationSelector: flags.Options.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      flags.Options.DisregardStatusWithLabelSelector,
		CIDR:                                  flags.Options.CIDR,
		NodeIP:                                flags.Options.NodeIP,
		NodeName:                              flags.Options.NodeName,
		NodePort:                              flags.Options.NodePort,
		PodPlayStageParallelism:               flags.Options.PodPlayStageParallelism,
		NodePlayStageParallelism:              flags.Options.NodePlayStageParallelism,
		NodeStages:                            nodeStages,
		PodStages:                             podStages,
		NodeLeaseParallelism:                  flags.Options.NodeLeaseParallelism,
		NodeLeaseDurationSeconds:              flags.Options.NodeLeaseDurationSeconds,
		ID:                                    id,
	})
	if err != nil {
		return err
	}

	serverAddress := flags.Options.ServerAddress
	if serverAddress == "" && flags.Options.NodePort != 0 {
		serverAddress = "0.0.0.0:" + format.String(flags.Options.NodePort)
	}

	if serverAddress != "" {
		clusterPortForwards := config.FilterWithTypeFromContext[*internalversion.ClusterPortForward](ctx)
		portForwards := config.FilterWithTypeFromContext[*internalversion.PortForward](ctx)
		clusterExecs := config.FilterWithTypeFromContext[*internalversion.ClusterExec](ctx)
		execs := config.FilterWithTypeFromContext[*internalversion.Exec](ctx)
		clusterLogs := config.FilterWithTypeFromContext[*internalversion.ClusterLogs](ctx)
		logs := config.FilterWithTypeFromContext[*internalversion.Logs](ctx)
		clusterAttaches := config.FilterWithTypeFromContext[*internalversion.ClusterAttach](ctx)
		attaches := config.FilterWithTypeFromContext[*internalversion.Attach](ctx)
		config := server.Config{
			TypedKwokClient:     typedKwokClient,
			EnableCRDs:          flags.Options.EnableCRDs,
			ClusterPortForwards: clusterPortForwards,
			PortForwards:        portForwards,
			ClusterExecs:        clusterExecs,
			Execs:               execs,
			ClusterLogs:         clusterLogs,
			Logs:                logs,
			ClusterAttaches:     clusterAttaches,
			Attaches:            attaches,
			Metrics:             metrics,
			Controller:          ctr,
		}
		svc, err := server.NewServer(config)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		err = svc.InstallMetrics()
		if err != nil {
			return fmt.Errorf("failed to install metrics: %w", err)
		}
		svc.InstallHealthz()

		if flags.Options.EnableDebuggingHandlers {
			svc.InstallDebuggingHandlers()
			svc.InstallProfilingHandler(flags.Options.EnableProfilingHandler, flags.Options.EnableContentionProfiling)
		} else {
			svc.InstallDebuggingDisabledHandlers()
		}
		go func() {
			err := svc.Run(ctx, serverAddress, flags.Options.TLSCertFile, flags.Options.TLSPrivateKeyFile)
			if err != nil {
				logger.Error("Failed to run server", err)
				os.Exit(1)
			}
		}()
	}

	err = ctr.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func filterStages(stages []*internalversion.Stage, apiGroup, kind string) []*internalversion.Stage {
	return slices.Filter(stages, func(stage *internalversion.Stage) bool {
		return stage.Spec.ResourceRef.APIGroup == apiGroup && stage.Spec.ResourceRef.Kind == kind
	})
}

func waitForReady(ctx context.Context, clientset kubernetes.Interface) error {
	logger := log.FromContext(ctx)
	backoff := wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   2,
		Jitter:   0.1,
		Steps:    5,
	}

	err := wait.Poll(ctx,
		func(ctx context.Context) (bool, error) {
			_, err := clientset.CoreV1().Nodes().List(ctx,
				metav1.ListOptions{
					Limit: 1,
				})
			if err != nil {
				logger.Error("Failed to list nodes", err)
				return false, nil
			}
			return true, nil
		},
		wait.WithExponentialBackoff(&backoff),
	)
	if err != nil {
		return err
	}
	return nil
}

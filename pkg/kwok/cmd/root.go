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
	"k8s.io/client-go/rest"
	"k8s.io/utils/clock"

	nodefast "sigs.k8s.io/kwok/kustomize/stage/node/fast"
	nodeheartbeat "sigs.k8s.io/kwok/kustomize/stage/node/heartbeat"
	nodeheartbeatwithlease "sigs.k8s.io/kwok/kustomize/stage/node/heartbeat-with-lease"
	podfast "sigs.k8s.io/kwok/kustomize/stage/pod/fast"
	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/apis/v1alpha1"
	"sigs.k8s.io/kwok/pkg/client/clientset/versioned"
	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/kwok/server"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/envs"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/version"
	"sigs.k8s.io/kwok/pkg/utils/wait"
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
	cmd.Flags().Var(&flags.Options.Manages, "manage", "Manages resources")
	cmd.Flags().StringVar(&flags.Options.ManageSingleNode, "manage-single-node", flags.Options.ManageSingleNode, "Node that matches the name will be watched and managed. It's conflicted with manage-nodes-with-annotation-selector, manage-nodes-with-label-selector and manage-all-nodes.")
	_ = cmd.Flags().MarkDeprecated("manage-single-node", "Please use --manage Node:metadata.name=<nodename> instead")
	cmd.Flags().BoolVar(&flags.Options.ManageAllNodes, "manage-all-nodes", flags.Options.ManageAllNodes, "All nodes will be watched and managed. It's conflicted with manage-nodes-with-annotation-selector, manage-nodes-with-label-selector and manage-single-node.")
	_ = cmd.Flags().MarkDeprecated("manage-all-nodes", "Please use --manage Node instead")
	cmd.Flags().StringVar(&flags.Options.ManageNodesWithAnnotationSelector, "manage-nodes-with-annotation-selector", flags.Options.ManageNodesWithAnnotationSelector, "Nodes that match the annotation selector will be watched and managed. It's conflicted with manage-all-nodes and manage-single-node.")
	_ = cmd.Flags().MarkDeprecated("manage-nodes-with-annotation-selector", "Please use --manage Node:metadata.annotations.<key>=<value> instead")
	cmd.Flags().StringVar(&flags.Options.ManageNodesWithLabelSelector, "manage-nodes-with-label-selector", flags.Options.ManageNodesWithLabelSelector, "Nodes that match the label selector will be watched and managed. It's conflicted with manage-all-nodes and manage-single-node.")
	_ = cmd.Flags().MarkDeprecated("manage-nodes-with-label-selector", "Please use --manage Node:metadata.labels.<key>=<value> instead")
	cmd.Flags().StringVar(&flags.Options.DisregardStatusWithAnnotationSelector, "disregard-status-with-annotation-selector", flags.Options.DisregardStatusWithAnnotationSelector, "All node/pod status excluding the ones that match the annotation selector will be watched and managed.")
	_ = cmd.Flags().MarkDeprecated("disregard-status-with-annotation-selector", "Please use Stage API instead")
	cmd.Flags().StringVar(&flags.Options.DisregardStatusWithLabelSelector, "disregard-status-with-label-selector", flags.Options.DisregardStatusWithLabelSelector, "All node/pod status excluding the ones that match the label selector will be watched and managed.")
	_ = cmd.Flags().MarkDeprecated("disregard-status-with-label-selector", "Please use Stage API instead")
	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", flags.Kubeconfig, "Path to the kubeconfig file to use")
	cmd.Flags().StringVar(&flags.Master, "master", flags.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	cmd.Flags().StringVar(&flags.Options.ServerAddress, "server-address", flags.Options.ServerAddress, "Address to expose the server on")
	cmd.Flags().UintVar(&flags.Options.NodeLeaseDurationSeconds, "node-lease-duration-seconds", flags.Options.NodeLeaseDurationSeconds, "Duration of node lease seconds")
	cmd.Flags().StringSliceVar(&flags.Options.EnableCRDs, "enable-crds", flags.Options.EnableCRDs, "List of CRDs to enable")

	cmd.Flags().BoolVar(&flags.Options.EnableCNI, "experimental-enable-cni", flags.Options.EnableCNI, "Experimental support for getting pod ip from CNI, for CNI-related components, Only works with Linux")
	_ = cmd.Flags().MarkDeprecated("experimental-enable-cni", "It will be removed and will be supported in the form of plugins")
	return cmd
}

var crdDefines = map[string]struct{}{
	v1alpha1.StageKind:                {},
	v1alpha1.AttachKind:               {},
	v1alpha1.ClusterAttachKind:        {},
	v1alpha1.ExecKind:                 {},
	v1alpha1.ClusterExecKind:          {},
	v1alpha1.PortForwardKind:          {},
	v1alpha1.ClusterPortForwardKind:   {},
	v1alpha1.LogsKind:                 {},
	v1alpha1.ClusterLogsKind:          {},
	v1alpha1.ResourceUsageKind:        {},
	v1alpha1.ClusterResourceUsageKind: {},
	v1alpha1.MetricKind:               {},
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

	for _, crd := range flags.Options.EnableCRDs {
		if _, ok := crdDefines[crd]; !ok {
			return fmt.Errorf("invalid crd: %s", crd)
		}
	}

	stagesData := config.FilterWithTypeFromContext[*internalversion.Stage](ctx)
	err := checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.StageKind, stagesData)
	if err != nil {
		return err
	}

	var groupStages map[internalversion.StageResourceRef][]*internalversion.Stage

	if !slices.Contains(flags.Options.EnableCRDs, v1alpha1.StageKind) {
		groupStages = slices.GroupBy(stagesData, func(stage *internalversion.Stage) internalversion.StageResourceRef {
			return stage.Spec.ResourceRef
		})

		nodeRef := internalversion.StageResourceRef{APIGroup: "v1", Kind: "Node"}
		podRef := internalversion.StageResourceRef{APIGroup: "v1", Kind: "Pod"}

		if len(groupStages[nodeRef]) == 0 {
			logger.Warn("No node stages found, using default node stages")
			groupStages[nodeRef], err = getDefaultNodeStages(flags.Options.NodeLeaseDurationSeconds == 0)
			if err != nil {
				return err
			}
		}

		if len(groupStages[podRef]) == 0 {
			groupStages[podRef], err = getDefaultPodStages()
			if err != nil {
				return err
			}
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

	restConfig, err := clientset.ToRESTConfig()
	if err != nil {
		return err
	}

	dynamicClient, err := clientset.ToDynamicClient()
	if err != nil {
		return err
	}

	impersonatingDynamicClient := clientset.ToImpersonatingDynamicClient()

	restMapper, err := clientset.ToRESTMapper()
	if err != nil {
		return err
	}

	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		return err
	}

	typedClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	typedKwokClient, err := versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	err = waitForReady(ctx, typedClient)
	if err != nil {
		return err
	}

	manages := flags.Options.Manages
	var nodeSel internalversion.ManageNodeSelector
	if len(manages) != 0 {
		nodeSel, err = manages.NodeSelector()
		if err != nil {
			return err
		}
	} else {
		switch {
		case flags.Options.ManageSingleNode != "":
			logger.Info("Watch single node",
				"node", flags.Options.ManageSingleNode,
			)
			nodeSel.ManageSingleNode = flags.Options.ManageSingleNode
		case flags.Options.ManageAllNodes:
			logger.Info("Watch all nodes")
			nodeSel.ManageAllNodes = true
		case flags.Options.ManageNodesWithAnnotationSelector != "" || flags.Options.ManageNodesWithLabelSelector != "":
			logger.Info("Watch nodes",
				"annotation", flags.Options.ManageNodesWithAnnotationSelector,
				"label", flags.Options.ManageNodesWithLabelSelector,
			)
			nodeSel.ManageNodesWithLabelSelector = flags.Options.ManageNodesWithLabelSelector
			nodeSel.ManageNodesWithAnnotationSelector = flags.Options.ManageNodesWithAnnotationSelector
		}
	}

	id, err := controllers.Identity()
	if err != nil {
		return err
	}
	ctx = log.NewContext(ctx, logger.With("id", id))

	metrics := config.FilterWithTypeFromContext[*internalversion.Metric](ctx)
	enableMetrics := len(metrics) != 0 || slices.Contains(flags.Options.EnableCRDs, v1alpha1.MetricKind)
	ctr, err := controllers.NewController(controllers.Config{
		Clock:                                 clock.RealClock{},
		DynamicClient:                         dynamicClient,
		RESTClient:                            restClient,
		RESTMapper:                            restMapper,
		ImpersonatingDynamicClient:            impersonatingDynamicClient,
		TypedClient:                           typedClient,
		TypedKwokClient:                       typedKwokClient,
		EnableCNI:                             flags.Options.EnableCNI,
		EnableMetrics:                         enableMetrics,
		EnablePodCache:                        enableMetrics,
		Manages:                               manages,
		NoManageNode:                          nodeSel.IsEmpty(),
		ManageSingleNode:                      nodeSel.ManageSingleNode,
		ManageAllNodes:                        nodeSel.ManageAllNodes,
		ManageNodesWithAnnotationSelector:     nodeSel.ManageNodesWithAnnotationSelector,
		ManageNodesWithLabelSelector:          nodeSel.ManageNodesWithLabelSelector,
		DisregardStatusWithAnnotationSelector: flags.Options.DisregardStatusWithAnnotationSelector,
		DisregardStatusWithLabelSelector:      flags.Options.DisregardStatusWithLabelSelector,
		CIDR:                                  flags.Options.CIDR,
		NodeIP:                                flags.Options.NodeIP,
		NodeName:                              flags.Options.NodeName,
		NodePort:                              flags.Options.NodePort,
		PodPlayStageParallelism:               flags.Options.PodPlayStageParallelism,
		NodePlayStageParallelism:              flags.Options.NodePlayStageParallelism,
		LocalStages:                           groupStages,
		NodeLeaseParallelism:                  flags.Options.NodeLeaseParallelism,
		NodeLeaseDurationSeconds:              flags.Options.NodeLeaseDurationSeconds,
		ID:                                    id,
	})
	if err != nil {
		return err
	}

	err = ctr.Start(ctx)
	if err != nil {
		return err
	}

	err = startServer(ctx, flags, ctr, typedKwokClient, nodeSel)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

func startServer(ctx context.Context, flags *flagpole, ctr *controllers.Controller, typedKwokClient versioned.Interface, nodeSelector internalversion.ManageNodeSelector) (err error) {
	logger := log.FromContext(ctx)

	serverAddress := flags.Options.ServerAddress
	if serverAddress == "" && flags.Options.NodePort != 0 {
		serverAddress = "0.0.0.0:" + format.String(flags.Options.NodePort)
	}

	if serverAddress != "" {
		mangeNode := !nodeSelector.IsEmpty()
		conf := server.Config{
			TypedKwokClient: typedKwokClient,
			NoManageNode:    nodeSelector.IsEmpty(),
			EnableCRDs:      flags.Options.EnableCRDs,
			DataSource:      ctr,
			NodeCacheGetter: ctr.GetNodeCache(),
			PodCacheGetter:  ctr.GetPodCache(),
		}

		if mangeNode {
			conf.ClusterPortForwards = config.FilterWithTypeFromContext[*internalversion.ClusterPortForward](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.ClusterPortForwardKind, conf.ClusterPortForwards)
			if err != nil {
				return err
			}

			conf.PortForwards = config.FilterWithTypeFromContext[*internalversion.PortForward](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.PortForwardKind, conf.PortForwards)
			if err != nil {
				return err
			}

			conf.ClusterExecs = config.FilterWithTypeFromContext[*internalversion.ClusterExec](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.ClusterExecKind, conf.ClusterExecs)
			if err != nil {
				return err
			}

			conf.Execs = config.FilterWithTypeFromContext[*internalversion.Exec](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.ExecKind, conf.Execs)
			if err != nil {
				return err
			}

			conf.ClusterLogs = config.FilterWithTypeFromContext[*internalversion.ClusterLogs](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.ClusterLogsKind, conf.ClusterLogs)
			if err != nil {
				return err
			}

			conf.Logs = config.FilterWithTypeFromContext[*internalversion.Logs](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.LogsKind, conf.Logs)
			if err != nil {
				return err
			}

			conf.ClusterAttaches = config.FilterWithTypeFromContext[*internalversion.ClusterAttach](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.ClusterAttachKind, conf.ClusterAttaches)
			if err != nil {
				return err
			}

			conf.Attaches = config.FilterWithTypeFromContext[*internalversion.Attach](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.AttachKind, conf.Attaches)
			if err != nil {
				return err
			}

			conf.ClusterResourceUsages = config.FilterWithTypeFromContext[*internalversion.ClusterResourceUsage](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.ClusterResourceUsageKind, conf.ClusterResourceUsages)
			if err != nil {
				return err
			}

			conf.ResourceUsages = config.FilterWithTypeFromContext[*internalversion.ResourceUsage](ctx)
			err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.ResourceUsageKind, conf.ResourceUsages)
			if err != nil {
				return err
			}
		}

		conf.Metrics = config.FilterWithTypeFromContext[*internalversion.Metric](ctx)
		err = checkConfigOrCRD(flags.Options.EnableCRDs, v1alpha1.MetricKind, conf.Metrics)
		if err != nil {
			return err
		}

		svc, err := server.NewServer(conf)
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}
		svc.InstallHealthz()

		if mangeNode {
			svc.InstallServiceDiscovery()
		}

		if mangeNode && flags.Options.EnableDebuggingHandlers {
			svc.InstallDebuggingHandlers()
			svc.InstallProfilingHandler(flags.Options.EnableProfilingHandler, flags.Options.EnableContentionProfiling)
		} else {
			svc.InstallDebuggingDisabledHandlers()
		}

		err = svc.InstallCRD(ctx)
		if err != nil {
			return fmt.Errorf("failed to install crd: %w", err)
		}

		err = svc.InstallMetrics(ctx)
		if err != nil {
			return fmt.Errorf("failed to install metrics: %w", err)
		}

		go func() {
			err := svc.Run(ctx, serverAddress, flags.Options.TLSCertFile, flags.Options.TLSPrivateKeyFile)
			if err != nil {
				// allow the server exit when work on host network
				podIP := envs.GetEnv("POD_IP", "")
				hostIP := envs.GetEnv("HOST_IP", "")
				if podIP == "" || hostIP == "" || podIP != hostIP {
					logger.Error("Failed to run server", err)
					os.Exit(1)
				} else {
					logger.Warn("Failed to run server, but allow the server exit when work on host network", "err", err)
				}
			}
		}()
	}

	<-ctx.Done()
	return nil
}

func checkConfigOrCRD[T metav1.Object](crds []string, kind string, crs []T) error {
	if slices.Contains(crds, kind) && len(crs) != 0 {
		return fmt.Errorf("%s already exists in --config, so please remove it, or remove %s from --enable-crd", kind, kind)
	}

	return nil
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

func getDefaultNodeStages(lease bool) ([]*internalversion.Stage, error) {
	nodeStages := []*internalversion.Stage{}
	nodeInitStage, err := config.UnmarshalWithType[*internalversion.Stage](nodefast.DefaultNodeInit)
	if err != nil {
		return nil, err
	}
	nodeStages = append(nodeStages, nodeInitStage)

	rawHeartbeat := nodeheartbeat.DefaultNodeHeartbeat
	if lease {
		rawHeartbeat = nodeheartbeatwithlease.DefaultNodeHeartbeatWithLease
	}

	nodeHeartbeatStage, err := config.UnmarshalWithType[*internalversion.Stage](rawHeartbeat)
	if err != nil {
		return nil, err
	}
	nodeStages = append(nodeStages, nodeHeartbeatStage)
	return nodeStages, nil
}

func getDefaultPodStages() ([]*internalversion.Stage, error) {
	return slices.MapWithError([]string{
		podfast.DefaultPodReady,
		podfast.DefaultPodComplete,
		podfast.DefaultPodDelete,
	}, config.UnmarshalWithType[*internalversion.Stage, string])
}

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

package cmd

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/flowcontrol"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwok/controllers"
	"sigs.k8s.io/kwok/pkg/kwok/controllers/templates"
	"sigs.k8s.io/kwok/pkg/log"
)

// NewCommand returns a new cobra.Command for root
func NewCommand() *cobra.Command {
	var (
		// The default IP assigned to the Pod on maintained Nodes.
		cidr = "10.0.0.1/24"
		// The ip of all nodes maintained by the Kwok
		nodeIP = net.ParseIP("196.168.0.1")
		// Default option to manage (i.e., maintain heartbeat/liveness of) all Nodes or not.
		manageAllNodes = false
		// Default annotations specified on Nodes to demand manage.
		// Note: when `all-node-manage` is specified as true, this is a no-op.
		manageNodesWithAnnotationSelector = ""
		// Default labels specified on Nodes to demand manage.
		// Note: when `all-node-manage` is specified as true, this is a no-op.
		manageNodesWithLabelSelector = ""
		// If a Node being managed has this annotation it will only keep its heartbeat not modify another status
		// If a Pod is on a managed Node and has this annotation status will not be modified
		disregardStatusWithAnnotationSelector = ""
		// If a Node being managed has this label it will only keep its heartbeat not modify another status
		// If a Pod is on a managed Node and has this label status will not be modified
		disregardStatusWithLabelSelector = ""

		serverAddress = ""
		master        = ""
		kubeconfig    = getEnv("KUBECONFIG", "")
	)

	cmd := &cobra.Command{
		Args:          cobra.NoArgs,
		Use:           "kwok [command]",
		Short:         "kwok is a tool for simulate thousands of fake kubelets",
		Long:          "kwok is a tool for simulate thousands of fake kubelets",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       consts.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if kubeconfig != "" {
				f, err := os.Stat(kubeconfig)
				if err != nil || f.IsDir() {
					kubeconfig = ""
				}
			}

			ctx := cmd.Context()
			logger := log.FromContext(ctx)

			clientset, err := newClientset(ctx, master, kubeconfig)
			if err != nil {
				return err
			}

			if manageAllNodes {
				if manageNodesWithAnnotationSelector != "" || manageNodesWithLabelSelector != "" {
					logger.Error("manage-all-nodes is conflicted with manage-nodes-with-annotation-selector and manage-nodes-with-label-selector.", nil)
					os.Exit(1)
				}
				logger.Info("Watch all nodes")
			} else if manageNodesWithAnnotationSelector != "" || manageNodesWithLabelSelector != "" {
				logger.Info("Watch nodes",
					"annotation", manageNodesWithAnnotationSelector,
					"label", manageNodesWithLabelSelector,
				)
			}

			backoff := wait.Backoff{
				Duration: 1 * time.Second,
				Factor:   2,
				Jitter:   0.1,
				Steps:    5,
			}
			err = wait.ExponentialBackoffWithContext(ctx, backoff,
				func() (bool, error) {
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
			)
			if err != nil {
				return err
			}

			ctr, err := controllers.NewController(controllers.Config{
				ClientSet:                             clientset,
				ManageAllNodes:                        manageAllNodes,
				ManageNodesWithAnnotationSelector:     manageNodesWithAnnotationSelector,
				ManageNodesWithLabelSelector:          manageNodesWithLabelSelector,
				DisregardStatusWithAnnotationSelector: disregardStatusWithAnnotationSelector,
				DisregardStatusWithLabelSelector:      disregardStatusWithLabelSelector,
				CIDR:                                  cidr,
				NodeIP:                                nodeIP.String(),
				PodStatusTemplate:                     templates.DefaultPodStatusTemplate,
				NodeHeartbeatTemplate:                 templates.DefaultNodeHeartbeatTemplate,
				NodeInitializationTemplate:            templates.DefaultNodeStatusTemplate,
			})
			if err != nil {
				return err
			}

			if serverAddress != "" {
				go Serve(ctx, serverAddress)
			}

			err = ctr.Start(ctx)
			if err != nil {
				return err
			}

			<-ctx.Done()
			return nil
		},
	}

	cmd.Flags().StringVar(&cidr, "cidr", cidr, "CIDR of the pod ip")
	cmd.Flags().IPVar(&nodeIP, "node-ip", nodeIP, "IP of the node")
	cmd.Flags().BoolVar(&manageAllNodes, "manage-all-nodes", manageAllNodes, "All nodes will be watched and managed. It's conflicted with manage-nodes-with-annotation-selector and manage-nodes-with-label-selector.")
	cmd.Flags().StringVar(&manageNodesWithAnnotationSelector, "manage-nodes-with-annotation-selector", manageNodesWithAnnotationSelector, "Nodes that match the annotation selector will be watched and managed. It's conflicted with manage-all-nodes.")
	cmd.Flags().StringVar(&manageNodesWithLabelSelector, "manage-nodes-with-label-selector", manageNodesWithLabelSelector, "Nodes that match the label selector will be watched and managed. It's conflicted with manage-all-nodes.")
	cmd.Flags().StringVar(&disregardStatusWithAnnotationSelector, "disregard-status-with-annotation-selector", disregardStatusWithAnnotationSelector, "All node/pod status excluding the ones that match the annotation selector will be watched and managed.")
	cmd.Flags().StringVar(&disregardStatusWithLabelSelector, "disregard-status-with-label-selector", disregardStatusWithLabelSelector, "All node/pod status excluding the ones that match the label selector will be watched and managed.")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to the kubeconfig file to use")
	cmd.Flags().StringVar(&master, "master", master, "Server is the address of the kubernetes cluster")
	cmd.Flags().StringVar(&serverAddress, "server-address", serverAddress, "Address to expose health and metrics on")

	return cmd
}

func Serve(ctx context.Context, address string) {
	promHandler := promhttp.Handler()
	svc := &http.Server{
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Addr: address,
		Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/healthz", "/health":
				rw.Write([]byte("health"))
			case "/metrics":
				promHandler.ServeHTTP(rw, r)
			default:
				http.NotFound(rw, r)
			}
		}),
	}

	logger := log.FromContext(ctx)
	err := svc.ListenAndServe()
	if err != nil {
		logger.Error("Fatal start server", err)
		os.Exit(1)
	}
}

// buildConfigFromFlags is a helper function that builds configs from a master url or a kubeconfig filepath.
func buildConfigFromFlags(ctx context.Context, masterUrl, kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" && masterUrl == "" {
		logger := log.FromContext(ctx)
		logger.Warn("Neither --kubeconfig nor --master was specified")
		logger.Info("Using the inClusterConfig")
		kubeconfig, err := rest.InClusterConfig()
		if err == nil {
			return kubeconfig, nil
		}
		logger.Error("Creating inClusterConfig", err)
		logger.Info("Falling back to default config")
	}
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{ClusterInfo: clientcmdapi.Cluster{Server: masterUrl}}).ClientConfig()
}

func newClientset(ctx context.Context, master, kubeconfig string) (kubernetes.Interface, error) {
	cfg, err := buildConfigFromFlags(ctx, master, kubeconfig)
	if err != nil {
		return nil, err
	}
	err = setConfigDefaults(cfg)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func setConfigDefaults(config *rest.Config) error {
	config.RateLimiter = flowcontrol.NewFakeAlwaysRateLimiter()
	return rest.SetKubernetesDefaults(config)
}

func getEnv(name string, defaults string) string {
	val, ok := os.LookupEnv(name)
	if ok {
		return val
	}
	return defaults
}

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

package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"sigs.k8s.io/kwok/pkg/controllers"
	"sigs.k8s.io/kwok/pkg/controllers/templates"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
)

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
	logger        = log.New(os.Stderr, "[kwok/controllers] ", log.LstdFlags)
)

func init() {
	pflag.StringVar(&cidr, "cidr", cidr, "CIDR of the pod ip")
	pflag.IPVar(&nodeIP, "node-ip", nodeIP, "IP of the node")
	pflag.BoolVar(&manageAllNodes, "manage-all-nodes", manageAllNodes, "All nodes will be watched and managed. It's conflicted with manage-nodes-with-annotation-selector and manage-nodes-with-label-selector.")
	pflag.StringVar(&manageNodesWithAnnotationSelector, "manage-nodes-with-annotation-selector", manageNodesWithAnnotationSelector, "Nodes that match the annotation selector will be watched and managed. It's conflicted with manage-all-nodes.")
	pflag.StringVar(&manageNodesWithLabelSelector, "manage-nodes-with-label-selector", manageNodesWithLabelSelector, "Nodes that match the label selector will be watched and managed. It's conflicted with manage-all-nodes.")
	pflag.StringVar(&disregardStatusWithAnnotationSelector, "disregard-status-with-annotation-selector", disregardStatusWithAnnotationSelector, "All node/pod status excluding the ones that match the annotation selector will be watched and managed.")
	pflag.StringVar(&disregardStatusWithLabelSelector, "disregard-status-with-label-selector", disregardStatusWithLabelSelector, "All node/pod status excluding the ones that match the label selector will be watched and managed.")
	pflag.StringVar(&kubeconfig, "kubeconfig", kubeconfig, "Path to the kubeconfig file to use")
	pflag.StringVar(&master, "master", master, "Server is the address of the kubernetes cluster")
	pflag.StringVar(&serverAddress, "server-address", serverAddress, "Address to expose health and metrics on")

	pflag.Parse()

	if kubeconfig != "" {
		f, err := os.Stat(kubeconfig)
		if err != nil || f.IsDir() {
			kubeconfig = ""
		}
	}
}

func main() {
	ctx := context.Background()

	clientset, err := newClientset(master, kubeconfig)
	if err != nil {
		logger.Fatalln(err)
	}

	if manageAllNodes {
		if manageNodesWithAnnotationSelector != "" || manageNodesWithLabelSelector != "" {
			logger.Fatalln("manage-all-nodes is conflicted with manage-nodes-with-annotation-selector and manage-nodes-with-label-selector.")
		}
		logger.Printf("Watch all nodes")
	} else if manageNodesWithAnnotationSelector != "" || manageNodesWithLabelSelector != "" {
		logger.Printf("Watch nodes with annotation %q and label %q", manageNodesWithAnnotationSelector, manageNodesWithLabelSelector)
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
				logger.Printf("Failed to list nodes: %v", err)
				return false, nil
			}
			return true, nil
		},
	)
	if err != nil {
		logger.Fatalf("Failed to list nodes: %v", err)
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
		Logger:                                logger,
		PodStatusTemplate:                     templates.DefaultPodStatusTemplate,
		NodeHeartbeatTemplate:                 templates.DefaultNodeHeartbeatTemplate,
		NodeInitializationTemplate:            templates.DefaultNodeStatusTemplate,
	})
	if err != nil {
		logger.Fatalln(err)
	}

	if serverAddress != "" {
		go Serve(ctx, serverAddress)
	}

	err = ctr.Start(ctx)
	if err != nil {
		logger.Fatalln(err)
	}

	<-ctx.Done()
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

	err := svc.ListenAndServe()
	if err != nil {
		logger.Fatal("Fatal start server")
	}
}

func newClientset(master, kubeconfig string) (kubernetes.Interface, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
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

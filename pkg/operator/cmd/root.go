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

// Package cmd defines a root command for the kwok.
package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"

	"sigs.k8s.io/kwok/pkg/client/clientset/versioned"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/operator/controllers"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/kubeconfig"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/version"
)

type flagpole struct {
	Kubeconfig string
	Master     string

	SourceControllerNamespace string
	SourceControllerName      string
}

// NewCommand returns a new cobra.Command for root
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}
	cmd := &cobra.Command{
		Args:          cobra.NoArgs,
		Use:           "kwok-operator",
		Short:         "kwok-operator is a operator for kwok-controller",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version.DisplayVersion(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runE(cmd.Context(), flags)
		},
	}

	flags.Kubeconfig = path.RelFromHome(kubeconfig.GetRecommendedKubeconfigPath())

	cmd.Flags().StringVar(&flags.Kubeconfig, "kubeconfig", flags.Kubeconfig, "Path to the kubeconfig file to use")
	cmd.Flags().StringVar(&flags.Master, "master", flags.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")

	cmd.Flags().StringVar(&flags.SourceControllerName, "controller-name", "kwok-controller", "The name of the controller to use for the source controller")
	cmd.Flags().StringVar(&flags.SourceControllerNamespace, "controller-namespace", "kube-system", "The namespace of the controller to use for the source controller")
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

	restConfig, err := clientset.ToRESTConfig()
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

	ctr, err := controllers.NewController(controllers.ControllerConfig{
		TypedClient:     typedClient,
		TypedKwokClient: typedKwokClient,
		SourceNamespace: flags.SourceControllerNamespace,
		SourceName:      flags.SourceControllerName,
	})
	if err != nil {
		return err
	}

	err = ctr.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}

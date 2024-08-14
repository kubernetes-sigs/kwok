/*
Copyright 2023 The Kubernetes Authors.

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

// Package del defines a command to delete data in etcd
package del

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/etcd"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/path"
)

type flagpole struct {
	Name      string
	Namespace string
	Output    string
}

// NewCommand returns a new cobra.Command for use hack the etcd data.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 2),
		Use:   "delete [resource] [name]",
		Short: "delete data in etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			err := runE(cmd.Context(), flags, args)
			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Output, "output", "o", "key", "output format. One of: (key, none).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	return cmd
}

func runE(ctx context.Context, flags *flagpole, args []string) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		return err
	}

	conf, err := rt.Config(ctx)
	if err != nil {
		return err
	}

	if rt.IsDryRun() {
		switch len(args) {
		case 1:
			if flags.Namespace == "" {
				dryrun.PrintMessage("kubectl delete %s --all -A", args[0])
			} else {
				dryrun.PrintMessage("kubectl delete %s --all -n %s", args[0], flags.Namespace)
			}
		case 2:
			if flags.Namespace == "" {
				dryrun.PrintMessage("kubectl delete %s %s", args[0], args[1])
			} else {
				dryrun.PrintMessage("kubectl delete %s %s -n %s", args[0], args[1], flags.Namespace)
			}
		default:
			if flags.Namespace == "" {
				dryrun.PrintMessage("kubectl delete all --all -A")
			} else {
				dryrun.PrintMessage("kubectl delete all -all -n %s", flags.Namespace)
			}
		}
		return nil
	}

	etcdclient, err := rt.GetEtcdClient(ctx)
	if err != nil {
		return err
	}

	var targetGvr schema.GroupVersionResource
	var targetName string
	var targetNamespace string
	if len(args) != 0 {
		kubeconfigPath := rt.GetWorkdirPath(runtime.InHostKubeconfigName)
		clientset, err := client.NewClientset("", kubeconfigPath)
		if err != nil {
			return err
		}

		dc, err := clientset.ToDiscoveryClient()
		if err != nil {
			return err
		}
		rl, err := dc.ServerPreferredResources()
		if err != nil {
			return err
		}

		resourceName := args[0]

		gvr, resource, err := client.MatchShortResourceName(rl, resourceName)
		if err != nil {
			return err
		}

		if resource.Namespaced {
			if flags.Namespace == "" {
				flags.Namespace = "default"
			}
		} else {
			if flags.Namespace != "" {
				return fmt.Errorf("resource %s is not namespaced", gvr)
			}
		}

		targetGvr = gvr
		targetNamespace = flags.Namespace
		if len(args) >= 2 {
			targetName = args[1]
		}
	}

	var count int
	var response func(kv *etcd.KeyValue) error
	if flags.Output == "key" {
		response = func(kv *etcd.KeyValue) error {
			count++
			_, _ = fmt.Fprintf(os.Stdout, "%s\n", kv.Key)
			return nil
		}
	}

	opOpts := []etcd.OpOption{
		etcd.WithName(targetName, targetNamespace),
		etcd.WithGVR(targetGvr),
	}

	if response != nil {
		opOpts = append(opOpts,
			etcd.WithKeysOnly(),
			etcd.WithResponse(response),
		)
	}

	err = etcdclient.Delete(ctx, conf.Options.EtcdPrefix,
		opOpts...,
	)
	if err != nil {
		return err
	}

	if log.IsTerminal() && flags.Output == "key" {
		fmt.Fprintf(os.Stderr, "delete %d keys\n", count)
	}
	return nil
}

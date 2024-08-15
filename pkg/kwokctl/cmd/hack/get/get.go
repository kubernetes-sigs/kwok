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

// Package get defines a command to get data in etcd
package get

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
	ChunkSize int64
	Watch     bool
	WatchOnly bool
}

// NewCommand returns a new cobra.Command for use hack the etcd data.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 2),
		Use:   "get [resource] [name]",
		Short: "get data in etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			err := runE(cmd.Context(), flags, args)
			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&flags.Output, "output", "o", "yaml", "output format. One of: (json, yaml, raw, key).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	cmd.Flags().BoolVarP(&flags.Watch, "watch", "w", false, "after listing/getting the requested object, watch for changes")
	cmd.Flags().BoolVar(&flags.WatchOnly, "watch-only", false, "watch for changes to the requested object(s), without listing/getting first")
	cmd.Flags().Int64Var(&flags.ChunkSize, "chunk-size", 500, "chunk size of the list pager")
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
				dryrun.PrintMessage("kubectl get %s --all -A", args[0])
			} else {
				dryrun.PrintMessage("kubectl get %s --all -n %s", args[0], flags.Namespace)
			}
		case 2:
			if flags.Namespace == "" {
				dryrun.PrintMessage("kubectl get %s %s", args[0], args[1])
			} else {
				dryrun.PrintMessage("kubectl get %s %s -n %s", args[0], args[1], flags.Namespace)
			}
		default:
			if flags.Namespace == "" {
				dryrun.PrintMessage("kubectl get all --all -A")
			} else {
				dryrun.PrintMessage("kubectl get all -all -n %s", flags.Namespace)
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

	switch flags.Output {
	case "json":
		outMediaType := etcd.JSONMediaType
		response = func(kv *etcd.KeyValue) error {
			count++
			value := kv.Value
			if value == nil {
				value = kv.PrevValue
			}
			inMediaType, err := etcd.DetectMediaType(value)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
				return nil
			}
			_, data, err := etcd.Convert(inMediaType, outMediaType, value)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "---\n# %s | %s\n%s\n", kv.Key, inMediaType, data)
			}
			return nil
		}
	case "yaml":
		outMediaType := etcd.YAMLMediaType
		response = func(kv *etcd.KeyValue) error {
			count++
			value := kv.Value
			if value == nil {
				value = kv.PrevValue
			}
			inMediaType, err := etcd.DetectMediaType(value)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
				return nil
			}
			_, data, err := etcd.Convert(inMediaType, outMediaType, value)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stdout, "---\n# %s | raw | %v\n# %s\n", kv.Key, err, value)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "---\n# %s | %s\n%s\n", kv.Key, inMediaType, data)
			}
			return nil
		}
	case "raw":
		response = func(kv *etcd.KeyValue) error {
			count++
			_, _ = fmt.Fprintf(os.Stdout, "%s\n%s\n", kv.Key, kv.Value)
			return nil
		}
	case "key":
		response = func(kv *etcd.KeyValue) error {
			count++
			_, _ = fmt.Fprintf(os.Stdout, "%s\n", kv.Key)
			return nil
		}
	default:
		return fmt.Errorf("unsupported output format: %s", flags.Output)
	}

	opOpts := []etcd.OpOption{
		etcd.WithName(targetName, targetNamespace),
		etcd.WithGVR(targetGvr),
		etcd.WithPageLimit(flags.ChunkSize),
		etcd.WithResponse(response),
	}

	if flags.Output == "key" {
		opOpts = append(opOpts,
			etcd.WithKeysOnly(),
		)
	}

	if flags.Watch {
		var rev int64
		if !flags.WatchOnly {
			rev, err = etcdclient.Get(ctx, conf.Options.EtcdPrefix,
				opOpts...,
			)
			if err != nil {
				return err
			}
		}

		opOpts = append(opOpts, etcd.WithRevision(rev))

		err = etcdclient.Watch(ctx, conf.Options.EtcdPrefix,
			opOpts...,
		)
		if err != nil {
			return err
		}
	} else {
		_, err = etcdclient.Get(ctx, conf.Options.EtcdPrefix,
			opOpts...,
		)
		if err != nil {
			return err
		}

		if log.IsTerminal() && flags.Output == "key" {
			fmt.Fprintf(os.Stderr, "get %d keys\n", count)
		}
	}
	return nil
}

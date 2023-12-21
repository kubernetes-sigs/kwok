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

// Package put defines a command to put data in etcd
package put

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/etcd"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

type flagpole struct {
	Name      string
	Namespace string
	Prefix    string
	Output    string
	Path      string
}

// NewCommand returns a new cobra.Command for use hack the etcd data.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.RangeArgs(0, 2),
		Use:   "put [resource] [name]",
		Short: "put data in etcd",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			err := runE(cmd.Context(), flags, args)
			if err != nil {
				return fmt.Errorf("%v: %w", args, err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flags.Prefix, "prefix", "/registry", "prefix of the key")
	cmd.Flags().StringVarP(&flags.Output, "output", "o", "key", "output format. One of: (key, none).")
	cmd.Flags().StringVarP(&flags.Namespace, "namespace", "n", "", "namespace of resource")
	cmd.Flags().StringVar(&flags.Path, "path", "", "path of the file")
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

	if rt.IsDryRun() {
		switch len(args) {
		case 1:
			if flags.Namespace == "" {
				dryrun.PrintMessage("cat %s | jq 'select(.kind==%q)' | kubectl apply -f -", flags.Path, args[0])
			} else {
				dryrun.PrintMessage("cat %s | jq 'select(.kind==%q)' | kubectl apply -f - -n %s", flags.Path, args[0], flags.Namespace)
			}
		case 2:
			if flags.Namespace == "" {
				dryrun.PrintMessage("cat %s | jq 'select(.kind==%q and .metadata.name==%q)' | kubectl apply -f -", flags.Path, args[0], args[1])
			} else {
				dryrun.PrintMessage("cat %s | jq 'select(.kind==%q and .metadata.name==%q)' | kubectl apply -f - -n %s", flags.Path, args[0], args[1], flags.Namespace)
			}
		default:
			if flags.Namespace == "" {
				dryrun.PrintMessage("kubectl apply -f %s", flags.Path)
			} else {
				dryrun.PrintMessage("kubectl apply -f %s -n %s", flags.Path, flags.Namespace)
			}
		}
		return nil
	}

	var reader io.Reader
	switch flags.Path {
	case "-":
		reader = os.Stdin
	case "":
		return fmt.Errorf("path is required")
	default:
		reader, err = os.Open(flags.Path)
		if err != nil {
			return err
		}
	}

	etcdclient, err := rt.GetEtcdClient(ctx)
	if err != nil {
		return err
	}

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

	var wantGVR *schema.GroupVersionResource
	var wantName string
	if len(args) != 0 {
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
				return fmt.Errorf("resource %s is not namespaced", resourceName)
			}
		}

		wantGVR = &gvr
		if len(args) >= 2 {
			wantName = args[1]
		}
	}

	start := time.Now()
	decoder := yaml.NewDecoder(reader)

	var count int
	var response func(kv *etcd.KeyValue) error
	if flags.Output == "key" {
		//nolint:unparam
		response = func(kv *etcd.KeyValue) error {
			count++
			if kv != nil {
				fmt.Fprintf(os.Stdout, "%s\n", kv.Key)
			}
			return nil
		}
	}

	err = decoder.DecodeToUnstructured(func(obj *unstructured.Unstructured) error {
		targetName := obj.GetName()
		if targetName == "" {
			return nil
		}

		targetGvr, resource, err := client.MatchGVK(rl, obj.GroupVersionKind())
		if err != nil {
			gvk := obj.GroupVersionKind()
			targetGvr, resource, err = client.MatchGK(rl, gvk.GroupKind())
			if err != nil {
				return err
			}
			logger.Warn("expected resource not found, no conversion to actual resource",
				"expected", gvk,
				"actual", targetGvr.GroupVersion().WithKind(gvk.Kind),
			)
		}

		targetNamespace := obj.GetNamespace()
		if resource.Namespaced {
			if targetNamespace == "" {
				targetNamespace = "default"
			}
		} else {
			if targetNamespace != "" {
				return fmt.Errorf("resource %s is not namespaced", targetGvr)
			}
		}

		if targetNamespace != "" && flags.Namespace != "" && targetNamespace != flags.Namespace {
			return nil
		}

		if wantGVR != nil && *wantGVR != targetGvr {
			return nil
		}

		if wantName != "" && wantName != targetName {
			return nil
		}

		if targetName == "" {
			logger.Warn("resource has no name", "resource", obj)
			return nil
		}

		mediaType, err := etcd.MediaTypeFromGVR(targetGvr)
		if err != nil {
			return err
		}

		t := obj.GetCreationTimestamp()
		if t.IsZero() {
			obj.SetCreationTimestamp(metav1.Time{Time: start})
		}

		obj.SetResourceVersion("")
		obj.SetSelfLink("")

		data, err := obj.MarshalJSON()
		if err != nil {
			return err
		}

		_, data, err = etcd.Convert(etcd.JSONMediaType, mediaType, data)
		if err != nil {
			return err
		}

		opOpts := []etcd.OpOption{
			etcd.WithName(targetName, targetNamespace),
			etcd.WithGVR(targetGvr),
		}

		if response != nil {
			opOpts = append(opOpts,
				etcd.WithResponse(response),
				etcd.WithKeysOnly(),
			)
		}

		err = etcdclient.Put(ctx, flags.Prefix, data,
			opOpts...,
		)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if log.IsTerminal() && flags.Output == "key" {
		fmt.Fprintf(os.Stderr, "put %d keys\n", count)
	}
	return nil
}

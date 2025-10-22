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

// Package diff provides a command to watch resource changes and generate diffs.
package diff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/kwokctl/etcd"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	clientset "sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/patch"
	"sigs.k8s.io/kwok/pkg/utils/path"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

type flagpole struct {
	Name      string
	Resources []string
	AllEvents bool
}

// NewCommand returns a new cobra.Command for watching resource changes and generating diffs.
func NewCommand(ctx context.Context) *cobra.Command {
	flags := &flagpole{}

	cmd := &cobra.Command{
		Args:  cobra.NoArgs,
		Use:   "diff",
		Short: "Watch resource changes and generate diffs",
		Long: `Watch for Kubernetes resource changes in the cluster and display diffs between the old and new states.
This is useful when writing Stage configurations to see what changes happen to resources.

Example:
  # Watch all resource changes
  kwokctl snapshot diff

  # Watch specific resources
  kwokctl snapshot diff --resources pods,nodes

  # Show all events including when resources are unchanged
  kwokctl snapshot diff --all-events
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags.Name = config.DefaultCluster
			return runE(cmd.Context(), flags)
		},
	}

	cmd.Flags().StringSliceVar(&flags.Resources, "resources", nil, "Specific resource types to watch (e.g., pods,nodes,services)")
	cmd.Flags().BoolVar(&flags.AllEvents, "all-events", false, "Show all events including when resources are unchanged")
	return cmd
}

func runE(ctx context.Context, flags *flagpole) error {
	name := config.ClusterName(flags.Name)
	workdir := path.Join(config.ClustersDir, flags.Name)

	logger := log.FromContext(ctx)
	logger = logger.With("cluster", flags.Name)
	ctx = log.NewContext(ctx, logger)

	rt, err := runtime.DefaultRegistry.Load(ctx, name, workdir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Warn("Cluster does not exist")
		}
		return err
	}

	conf, err := rt.Config(ctx)
	if err != nil {
		return err
	}

	etcdclient, cancel, err := rt.GetEtcdClient(ctx)
	if err != nil {
		return err
	}
	defer cancel()

	clientset, err := rt.GetClientset(ctx)
	if err != nil {
		return err
	}

	watcher, err := newDiffWatcher(clientset, etcdclient, conf.Options.EtcdPrefix, flags)
	if err != nil {
		return err
	}

	logger.Info("Watching for resource changes...")
	logger.Info("Press Ctrl+C to stop watching")

	return watcher.watch(ctx)
}

type diffWatcher struct {
	clientset       clientset.Clientset
	etcdClient      etcd.Client
	prefix          string
	restMapper      meta.RESTMapper
	patchMetaSchema *patch.PatchMetaFromOpenAPI3
	track           map[log.ObjectRef]json.RawMessage
	flags           *flagpole
	resourceFilter  map[string]bool
}

func newDiffWatcher(cs clientset.Clientset, etcdClient etcd.Client, prefix string, flags *flagpole) (*diffWatcher, error) {
	restMapper, err := cs.ToRESTMapper()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest mapper: %w", err)
	}

	restConfig, err := cs.ToRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get rest config: %w", err)
	}

	restConfig.GroupVersion = &schema.GroupVersion{}
	restClient, err := rest.RESTClientFor(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest client: %w", err)
	}

	patchMetaSchema := patch.NewPatchMetaFromOpenAPI3(restClient)

	resourceFilter := make(map[string]bool)
	if len(flags.Resources) > 0 {
		for _, r := range flags.Resources {
			resourceFilter[strings.ToLower(r)] = true
		}
	}

	return &diffWatcher{
		clientset:       cs,
		etcdClient:      etcdClient,
		prefix:          prefix,
		restMapper:      restMapper,
		patchMetaSchema: patchMetaSchema,
		track:           make(map[log.ObjectRef]json.RawMessage),
		flags:           flags,
		resourceFilter:  resourceFilter,
	}, nil
}

func (w *diffWatcher) watch(ctx context.Context) error {
	logger := log.FromContext(ctx)

	// First, save the initial state
	logger.Info("Loading initial state...")
	rev, err := w.etcdClient.Get(ctx, w.prefix,
		etcd.WithResponse(func(kv *etcd.KeyValue) error {
			return w.saveInitialState(kv)
		}),
	)
	if err != nil {
		return err
	}

	logger.Info("Initial state loaded, watching for changes...")

	// Now watch for changes
	return w.etcdClient.Watch(ctx, w.prefix,
		etcd.WithRevision(rev),
		etcd.WithResponse(func(kv *etcd.KeyValue) error {
			return w.handleChange(ctx, kv)
		}),
	)
}

func (w *diffWatcher) saveInitialState(kv *etcd.KeyValue) error {
	value := kv.Value
	if value == nil {
		return nil
	}

	inMediaType, err := etcd.DetectMediaType(value)
	if err != nil {
		return nil // Skip invalid data
	}

	_, data, err := etcd.Convert(inMediaType, etcd.JSONMediaType, value)
	if err != nil {
		return nil // Skip conversion errors
	}

	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(data)
	if err != nil {
		return nil // Skip unmarshaling errors
	}

	if obj.GetName() == "" {
		return nil
	}

	// Check resource filter
	if !w.shouldWatch(obj) {
		return nil
	}

	w.track[log.KObj(obj)] = data
	return nil
}

func (w *diffWatcher) shouldWatch(obj *unstructured.Unstructured) bool {
	if len(w.resourceFilter) == 0 {
		return true
	}

	kind := strings.ToLower(obj.GetKind())
	return w.resourceFilter[kind]
}

func (w *diffWatcher) handleChange(ctx context.Context, kv *etcd.KeyValue) error {
	logger := log.FromContext(ctx)

	lastValue := kv.Value
	if lastValue == nil {
		lastValue = kv.PrevValue
	}
	if lastValue == nil {
		return nil
	}

	inMediaType, err := etcd.DetectMediaType(lastValue)
	if err != nil {
		return nil // Skip invalid data
	}

	_, data, err := etcd.Convert(inMediaType, etcd.JSONMediaType, lastValue)
	if err != nil {
		return nil // Skip conversion errors
	}

	obj := &unstructured.Unstructured{}
	err = obj.UnmarshalJSON(data)
	if err != nil {
		return nil // Skip unmarshaling errors
	}

	if obj.GetName() == "" {
		return nil
	}

	// Check resource filter
	if !w.shouldWatch(obj) {
		return nil
	}

	key := log.KObj(obj)
	gvk := obj.GroupVersionKind()
	gk := gvk.GroupKind()

	restMapping, err := w.restMapper.RESTMapping(gk)
	if err != nil {
		return nil // Skip resources we can't map
	}

	resourceName := fmt.Sprintf("%s/%s", obj.GetKind(), obj.GetName())
	if obj.GetNamespace() != "" {
		resourceName = fmt.Sprintf("%s/%s/%s", obj.GetKind(), obj.GetNamespace(), obj.GetName())
	}

	switch {
	case kv.Value != nil:
		obj.SetResourceVersion("")
		modified, err := json.Marshal(obj)
		if err != nil {
			return nil
		}

		original, ok := w.track[key]
		if !ok {
			// New resource created
			w.track[key] = modified
			logger.Info("Resource created", "resource", resourceName)
			if w.flags.AllEvents {
				w.printYAML(ctx, obj)
			}
			return nil
		}

		// Resource modified, generate and display diff
		gvr := restMapping.Resource
		gvr.Version = gvk.Version

		patchMeta, err := w.patchMetaSchema.Lookup(gvr)
		if err != nil {
			// Fallback to struct-based patch meta
			s, err0 := etcd.PatchMetaFromStruct(restMapping.GroupVersionKind)
			if err0 != nil {
				return nil // Skip if we can't get patch meta
			}
			patchMeta = s
		}

		patchData, err := strategicpatch.CreateTwoWayMergePatchUsingLookupPatchMeta(original, modified, patchMeta)
		if err != nil {
			return nil // Skip patch creation errors
		}

		// Check if there's an actual change
		if string(patchData) == "{}" {
			if w.flags.AllEvents {
				logger.Info("Resource unchanged", "resource", resourceName)
			}
			return nil
		}

		w.track[key] = modified

		// Display the diff
		logger.Info("Resource modified", "resource", resourceName)
		fmt.Printf("\n=== %s ===\n", resourceName)
		fmt.Println("Patch:")
		w.printPatchAsYAML(patchData)
		fmt.Println()

		return nil

	default:
		// Resource deleted
		delete(w.track, key)
		logger.Info("Resource deleted", "resource", resourceName)
		if w.flags.AllEvents {
			w.printYAML(ctx, obj)
		}
		return nil
	}
}

func (w *diffWatcher) printYAML(ctx context.Context, obj *unstructured.Unstructured) {
	data, err := yaml.Marshal(obj)
	if err != nil {
		log.FromContext(ctx).Error("Failed to marshal object", err)
		return
	}
	fmt.Println(string(data))
}

func (w *diffWatcher) printPatchAsYAML(patchData []byte) {
	var patch map[string]interface{}
	if err := json.Unmarshal(patchData, &patch); err != nil {
		fmt.Println(string(patchData))
		return
	}

	data, err := yaml.Marshal(patch)
	if err != nil {
		fmt.Println(string(patchData))
		return
	}

	// Add indentation for better readability
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Printf("  %s\n", line)
		}
	}
}

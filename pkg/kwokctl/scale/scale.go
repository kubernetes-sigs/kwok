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

package scale

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/pager"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/snapshot"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/client"
	"sigs.k8s.io/kwok/pkg/utils/gotpl"
	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// Config is the configuration for scaling a resource.
type Config struct {
	Parameters   any
	Template     string
	Name         string
	Namespace    string
	Replicas     int
	SerialLength int
	DryRun       bool
}

// Scale scales a resource in a cluster.
func Scale(ctx context.Context, clientset client.Clientset, conf Config) error {
	if conf.SerialLength == 0 && conf.Replicas > 1 {
		return fmt.Errorf("serial length must be greater than 0 when replicas is greater than 1")
	}

	param := conf.Parameters

	name := conf.Name
	namespace := conf.Namespace
	index := 0
	renderer := gotpl.NewRenderer(gotpl.FuncMap{
		"Name": func() string {
			return name
		},
		"Namespace": func() string {
			return namespace
		},
		"Index": func() int {
			return index
		},
		"AddCIDR": utilsnet.AddCIDR,
	})
	data, err := renderer.ToJSON(conf.Template, param)
	if err != nil {
		return err
	}

	if conf.DryRun {
		dryrun.PrintMessage("# Scale resource %s to %d replicas", conf.Name, conf.Replicas)
		dryrun.PrintMessage("# Resource example: %s", string(data))
		return nil
	}

	var u *unstructured.Unstructured
	err = json.Unmarshal(data, &u)
	if err != nil {
		return err
	}
	gv, err := schema.ParseGroupVersion(u.GetAPIVersion())
	if err != nil {
		return err
	}

	dynamicClient, err := clientset.ToDynamicClient()
	if err != nil {
		return err
	}

	restMapper, err := clientset.ToRESTMapper()
	if err != nil {
		return err
	}
	gvr, err := restMapper.ResourceFor(schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: u.GetKind(),
	})
	if err != nil {
		return err
	}

	nri := dynamicClient.Resource(gvr)

	logger := log.FromContext(ctx)
	logger = logger.With("name", conf.Name, "replicas", conf.Replicas, "resource", gvr.Resource)

	var ri dynamic.ResourceInterface = nri

	if namespace == "" {
		namespace = u.GetNamespace()
	}
	if namespace != "" {
		ri = nri.Namespace(namespace)
		logger = logger.With("namespace", namespace)
	}

	start := time.Now()
	listPager := pager.New(func(ctx context.Context, opts metav1.ListOptions) (apiruntime.Object, error) {
		return ri.List(ctx, opts)
	})
	deleteCount := 0
	objs := make([]softInfo, 0, conf.Replicas)
	sorted := false
	err = listPager.EachListItem(ctx, metav1.ListOptions{
		LabelSelector: labelNameKey + "=" + conf.Name,
	}, func(raw apiruntime.Object) error {
		obj := raw.(*unstructured.Unstructured)

		// If list is not full, append it.
		if len(objs) < cap(objs) {
			objs = append(objs, softInfo{
				Name:              obj.GetName(),
				CreationTimestamp: obj.GetCreationTimestamp(),
			})
			return nil
		}

		// List is full, sort it.
		if !sorted {
			sort.Slice(objs, func(i, j int) bool {
				itime := objs[i].CreationTimestamp
				jtime := objs[j].CreationTimestamp
				return itime.Before(&jtime)
			})
			sorted = true
		}

		deleteCount++

		if len(objs) == 0 {
			err = ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
			if err != nil {
				logger.Error("Delete resource", err)
			}
			return nil
		}

		// New object is newer than the end object, delete the new object.
		endObj := objs[len(objs)-1]
		if endObj.Less(obj.GetCreationTimestamp(), obj.GetName()) {
			// Delete the last object.
			err = ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
			if err != nil {
				logger.Error("Delete resource", err)
			}
			return nil
		}

		// Delete the end object.
		err = ri.Delete(ctx, endObj.Name, metav1.DeleteOptions{})
		if err != nil {
			logger.Error("Delete resource", err)
		}

		// Find the index of the new object to be inserted.
		index, _ := sort.Find(len(objs), func(i int) int {
			if objs[i].Less(obj.GetCreationTimestamp(), obj.GetName()) {
				return -1
			}
			return 1
		})

		if index == len(objs) {
			// Delete the last object.
			err = ri.Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
			if err != nil {
				logger.Error("Delete resource", err)
			}
			return nil
		}
		// Insert the new object.
		copy(objs[index+1:], objs[index:len(objs)-1])
		objs[index] = softInfo{
			Name:              obj.GetName(),
			CreationTimestamp: obj.GetCreationTimestamp(),
		}
		return nil
	})
	if err != nil {
		return err
	}

	if deleteCount > 0 {
		logger.Info("Deleted resources", "counter", deleteCount, "elapsed", time.Since(start))
		return nil
	}

	if len(objs) == cap(objs) {
		logger.Info("Nothing to do")
		return nil
	}

	has := map[string]struct{}{}
	for _, item := range objs {
		has[item.Name] = struct{}{}
	}

	wantCreate := conf.Replicas - len(objs)

	// free memory
	objs = nil

	buf := bytes.NewBuffer(nil)
	gen := newResourceGenerator(func(_ int) ([]byte, error) {
		for {
			name = generateSerialNumber(conf.Name, index, conf.SerialLength)
			_, ok := has[name]
			if !ok {
				break
			}
			index++
		}
		// defer to increment index
		defer func() {
			index++
		}()

		data, err := renderer.ToJSON(conf.Template, param)
		if err != nil {
			return nil, err
		}

		var u *unstructured.Unstructured
		err = json.Unmarshal(data, &u)
		if err != nil {
			return nil, err
		}

		labels := u.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels[labelNameKey] = conf.Name
		u.SetLabels(labels)
		u.SetNamespace(namespace)
		u.SetName(name)

		buf.Reset()
		_, _ = buf.WriteString("---\n")
		encoder := yaml.NewEncoder(buf)
		err = encoder.Encode(u)
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}, wantCreate)

	ctx = log.NewContext(ctx, logger)

	loader, err := snapshot.NewLoader(snapshot.LoadConfig{
		Clientset: clientset,
		NoFilers:  true,
	})
	if err != nil {
		return err
	}

	decoder := yaml.NewDecoder(gen)
	err = loader.Load(ctx, decoder)
	if err != nil {
		return err
	}

	return nil
}

var (
	labelNameKey = "kwok.x-k8s.io/kwokctl-scale"
)

type softInfo struct {
	Name              string
	CreationTimestamp metav1.Time
}

func (s softInfo) Less(creationTimestamp metav1.Time, name string) bool {
	cmp := s.CreationTimestamp.Time.Compare(creationTimestamp.Time)
	if cmp == 0 {
		return s.Name < name
	}

	return cmp < 0
}

type resourceGenerator struct {
	counter     int
	index       int
	dataGenFunc func(index int) ([]byte, error)
	buf         []byte
}

func newResourceGenerator(dataGenFunc func(index int) ([]byte, error), counter int) *resourceGenerator {
	return &resourceGenerator{
		counter:     counter,
		dataGenFunc: dataGenFunc,
	}
}

func (g *resourceGenerator) Read(p []byte) (n int, err error) {
	if len(g.buf) == 0 && g.counter == g.index {
		return 0, io.EOF
	}
	if len(g.buf) == 0 {
		buf, err := g.dataGenFunc(g.index)
		if err != nil {
			return 0, err
		}
		g.buf = buf
		g.index++
	}
	n = copy(p, g.buf)
	g.buf = g.buf[n:]
	return n, nil
}

func generateSerialNumber(name string, n int, minLen int) string {
	if minLen == 0 {
		return name
	}
	return fmt.Sprintf("%s-%0*d", name, minLen, n)
}

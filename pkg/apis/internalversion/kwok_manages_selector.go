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

package internalversion

import (
	"fmt"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/selection"

	"sigs.k8s.io/kwok/pkg/utils/maps"
	"sigs.k8s.io/kwok/pkg/utils/slices"
)

// ManagesSelectors holds information about the manages selectors.
type ManagesSelectors []ManagesSelector

// ManagesSelector holds information about the manages selector.
type ManagesSelector struct {
	// Kind of the referent.
	Kind string
	// Group of the referent.
	Group string
	// Version of the referent.
	Version string
	// Namespace of the referent. only valid if it is a namespaced.
	Namespace string
	// Name of the referent. specify only this one.
	Name string
	// Labels of the referent. specify matched with labels.
	Labels map[string]string
	// Annotations of the referent. specify matched with annotations.
	Annotations map[string]string
}

func (s *ManagesSelectors) Set(sel string) error {
	p, err := parseManagesSelector(sel)
	if err != nil {
		return err
	}
	*s = append(*s, *p)
	return nil
}

func (s ManagesSelectors) Type() string {
	return "ManagesSelectorSlice"
}

func (s ManagesSelectors) String() string {
	strSlice := slices.Map(s, func(t ManagesSelector) string {
		return t.String()
	})
	return strings.Join(strSlice, " ")
}

func (s *ManagesSelector) Set(sel string) error {
	p, err := parseManagesSelector(sel)
	if err != nil {
		return err
	}
	*s = *p
	return nil
}

func (s *ManagesSelector) Type() string {
	return "ManagesSelector"
}

func parseManagesSelector(arg string) (*ManagesSelector, error) {
	items := strings.Split(arg, ":")

	t := ManagesSelector{}
	gvk := items[0]
	if gvk == "" {
		return nil, fmt.Errorf("invalid empty target resource ref")
	}

	sepVersion := strings.Index(gvk, "/")
	if sepVersion != -1 {
		t.Version = gvk[sepVersion+1:]
		gvk = gvk[:sepVersion]
	}

	sepGroup := strings.Index(gvk, ".")
	if sepGroup != -1 {
		t.Kind = gvk[:sepGroup]
		t.Group = gvk[sepGroup+1:]
	} else {
		t.Kind = gvk
	}

	for _, item := range items[1:] {
		sel, err := fields.ParseSelector(item)
		if err != nil {
			return nil, err
		}
		for _, req := range sel.Requirements() {
			if req.Operator != selection.Equals && req.Operator != selection.DoubleEquals {
				return nil, fmt.Errorf("invalid selector requirements: %s", req.Operator)
			}
			switch req.Field {
			case "metadata.name":
				t.Name = req.Value
			case "metadata.namespace":
				t.Namespace = req.Value
			default:
				sp := strings.SplitN(req.Field, ".", 3)
				if len(sp) < 2 {
					return nil, fmt.Errorf("error target resource ref: %s", item)
				}
				if sp[0] != "metadata" {
					return nil, fmt.Errorf("error target resource ref: %s", item)
				}

				switch sp[1] {
				case "labels":
					if t.Labels == nil {
						t.Labels = map[string]string{}
					}
					t.Labels[sp[2]] = req.Value
				case "annotations":
					if t.Annotations == nil {
						t.Annotations = map[string]string{}
					}
					t.Annotations[sp[2]] = req.Value
				default:
					return nil, fmt.Errorf("error target resource ref: %s", item)
				}
			}
		}
	}
	return &t, nil
}

func (s *ManagesSelector) String() string {
	if s == nil {
		return ""
	}

	buf := &strings.Builder{}
	buf.WriteString(s.Kind)
	if s.Group != "" {
		buf.WriteString(fmt.Sprintf(".%s", s.Group))
	}
	if s.Version != "" {
		buf.WriteString(fmt.Sprintf("/%s", s.Version))
	}
	if s.Name != "" {
		buf.WriteString(fmt.Sprintf(":metadata.name=%s", s.Name))
	}
	if s.Namespace != "" {
		buf.WriteString(fmt.Sprintf(":metadata.namespace=%s", s.Namespace))
	}
	if len(s.Labels) > 0 {
		keys := maps.Keys(s.Labels)
		sort.Strings(keys)
		for _, k := range keys {
			buf.WriteString(fmt.Sprintf(":metadata.labels.%s=%s", k, s.Labels[k]))
		}
	}
	if len(s.Annotations) > 0 {
		keys := maps.Keys(s.Annotations)
		sort.Strings(keys)
		for _, k := range keys {
			buf.WriteString(fmt.Sprintf(":metadata.annotations.%s=%s", k, s.Annotations[k]))
		}
	}
	return buf.String()
}

func (s ManagesSelectors) MatchStage(stage *Stage) bool {
	for _, t := range s {
		if t.MatchStage(stage) {
			return true
		}
	}
	return false
}

func (s *ManagesSelector) MatchStage(stage *Stage) bool {
	spec := stage.Spec
	rr := spec.ResourceRef

	if s.Kind != rr.Kind {
		return false
	}

	gv := schema.GroupVersion{
		Group:   s.Group,
		Version: s.Version,
	}
	apiGroup := gv.String()
	if apiGroup == "" {
		apiGroup = "v1"
	}

	if rr.APIGroup == "" {
		rr.APIGroup = "v1"
	}

	if apiGroup != rr.APIGroup {
		return false
	}

	if spec.Selector != nil {
		if len(s.Labels) != 0 {
			ml := spec.Selector.MatchLabels
			for k, v := range s.Labels {
				if mv, ok := ml[k]; ok && mv != v {
					return false
				}
			}
		}
		if len(s.Annotations) != 0 {
			ma := spec.Selector.MatchAnnotations
			for k, v := range s.Annotations {
				if mv, ok := ma[k]; ok && mv != v {
					return false
				}
			}
		}
	}

	return true
}

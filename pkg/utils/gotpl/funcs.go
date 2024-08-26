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

package gotpl

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/utils/expression"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

var (
	// genericFuncs is generic template functions.
	genericFuncs = sprig.TxtFuncMap()
)

func init() {
	genericFuncs["GOOS"] = func() string {
		return runtime.GOOS
	}
	genericFuncs["GOARCH"] = func() string {
		return runtime.GOARCH
	}
}

var (
	startTime = time.Now().Format(time.RFC3339Nano)

	defaultFuncs = FuncMap{
		"Quote": func(s any) string {
			data, err := json.Marshal(s)
			if err != nil {
				return strconv.Quote(fmt.Sprint(s))
			}
			if len(data) == 0 {
				return `""`
			}
			if data[0] == '"' {
				return string(data)
			}
			return strconv.Quote(string(data))
		},
		"Now": func() string {
			return time.Now().Format(time.RFC3339Nano)
		},
		"StartTime": func() string {
			return startTime
		},
		"YAML": func(s interface{}, indent ...int) (string, error) {
			d, err := yaml.Marshal(s)
			if err != nil {
				return "", err
			}

			data := string(d)
			if len(indent) == 1 && indent[0] > 0 {
				pad := strings.Repeat(" ", indent[0]*2)
				data = strings.ReplaceAll("\n"+data, "\n", "\n"+pad)
			}
			return data, nil
		},
		"Version": func() string {
			return consts.Version
		},

		"NodeConditions": func() interface{} {
			return nodeConditionsData
		},
	}

	// https://kubernetes.io/docs/concepts/architecture/nodes/#condition
	nodeConditions = []corev1.NodeCondition{
		{
			Type:    corev1.NodeReady,
			Status:  corev1.ConditionTrue,
			Reason:  "KubeletReady",
			Message: "kubelet is posting ready status",
		},
		{
			Type:    corev1.NodeMemoryPressure,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeletHasSufficientMemory",
			Message: "kubelet has sufficient memory available",
		},
		{
			Type:    corev1.NodeDiskPressure,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeletHasNoDiskPressure",
			Message: "kubelet has no disk pressure",
		},
		{
			Type:    corev1.NodePIDPressure,
			Status:  corev1.ConditionFalse,
			Reason:  "KubeletHasSufficientPID",
			Message: "kubelet has sufficient PID available",
		},
		{
			Type:    corev1.NodeNetworkUnavailable,
			Status:  corev1.ConditionFalse,
			Reason:  "RouteCreated",
			Message: "RouteController created a route",
		},
	}
	nodeConditionsData, _ = expression.ToJSONStandard(nodeConditions)
)

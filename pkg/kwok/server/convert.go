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

package server

import (
	"net/url"

	"github.com/emicklei/go-restful/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type portForwardRequestParams struct {
	podNamespace string
	podName      string
	podUID       types.UID
}

func getPortForwardRequestParams(req *restful.Request) portForwardRequestParams {
	return portForwardRequestParams{
		podNamespace: req.PathParameter("podNamespace"),
		podName:      req.PathParameter("podID"),
		podUID:       types.UID(req.PathParameter("uid")),
	}
}

type execRequestParams struct {
	podNamespace  string
	podName       string
	podUID        types.UID
	containerName string
	cmd           []string
}

func getExecRequestParams(req *restful.Request) execRequestParams {
	return execRequestParams{
		podNamespace:  req.PathParameter("podNamespace"),
		podName:       req.PathParameter("podID"),
		podUID:        types.UID(req.PathParameter("uid")),
		containerName: req.PathParameter("containerName"),
		cmd:           req.Request.URL.Query()["command"],
	}
}

//nolint:revive
func convert_url_Values_To_v1_PodLogOptions(in *url.Values, out *corev1.PodLogOptions, s conversion.Scope) error {
	if values, ok := map[string][]string(*in)["container"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_string(&values, &out.Container, s); err != nil {
			return err
		}
	} else {
		out.Container = ""
	}
	if values, ok := map[string][]string(*in)["follow"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_bool(&values, &out.Follow, s); err != nil {
			return err
		}
	} else {
		out.Follow = false
	}
	if values, ok := map[string][]string(*in)["previous"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_bool(&values, &out.Previous, s); err != nil {
			return err
		}
	} else {
		out.Previous = false
	}
	if values, ok := map[string][]string(*in)["sinceSeconds"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_Pointer_int64(&values, &out.SinceSeconds, s); err != nil {
			return err
		}
	} else {
		out.SinceSeconds = nil
	}
	if values, ok := map[string][]string(*in)["sinceTime"]; ok && len(values) > 0 {
		if err := metav1.Convert_Slice_string_To_Pointer_v1_Time(&values, &out.SinceTime, s); err != nil {
			return err
		}
	} else {
		out.SinceTime = nil
	}
	if values, ok := map[string][]string(*in)["timestamps"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_bool(&values, &out.Timestamps, s); err != nil {
			return err
		}
	} else {
		out.Timestamps = false
	}
	if values, ok := map[string][]string(*in)["tailLines"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_Pointer_int64(&values, &out.TailLines, s); err != nil {
			return err
		}
	} else {
		out.TailLines = nil
	}
	if values, ok := map[string][]string(*in)["limitBytes"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_Pointer_int64(&values, &out.LimitBytes, s); err != nil {
			return err
		}
	} else {
		out.LimitBytes = nil
	}
	if values, ok := map[string][]string(*in)["insecureSkipTLSVerifyBackend"]; ok && len(values) > 0 {
		if err := runtime.Convert_Slice_string_To_bool(&values, &out.InsecureSkipTLSVerifyBackend, s); err != nil {
			return err
		}
	} else {
		out.InsecureSkipTLSVerifyBackend = false
	}
	return nil
}

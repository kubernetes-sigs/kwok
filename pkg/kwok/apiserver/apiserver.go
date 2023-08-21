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

package apiserver

import (
	"github.com/emicklei/go-restful/v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	cminstall "k8s.io/metrics/pkg/apis/custom_metrics/install"
	eminstall "k8s.io/metrics/pkg/apis/external_metrics/install"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/apiserver/installer"
)

var (
	scheme         = runtime.NewScheme()
	codecs         = serializer.NewCodecFactory(scheme)
	parameterCodec = runtime.NewParameterCodec(scheme)
)

func init() {
	cminstall.Install(scheme)
	eminstall.Install(scheme)
	utilruntime.Must(installer.RegisterConversions(scheme))
}

// InstallRootAPIs installs the root APIs for the apiserver.
func InstallRootAPIs(container *restful.Container) discovery.GroupManager {
	handler := discovery.NewRootAPIsHandler(discovery.CIDRRule{}, codecs)
	container.Handle(discovery.APIGroupPrefix, handler)
	return handler
}

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

package internalversion

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PortForward provides port forward configuration for a single pod.
type PortForward struct {
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	metav1.ObjectMeta
	// Spec holds spec for port forward.
	Spec PortForwardSpec
}

// PortForwardSpec holds spec for port forward.
type PortForwardSpec struct {
	// Forward is a list of forwards to configure.
	Forwards []Forward
}

// Forward holds information how to forward based on ports.
type Forward struct {
	// Ports is a list of ports to forward.
	// if not set, all ports will be forwarded.
	Ports []int32
	// Target is the target to forward to.
	Target *ForwardTarget
	// Command is the command to run to forward with stdin/stdout.
	// if set, Target will be ignored.
	Command []string
	// HTTPRoutes defines a list of predefined HTTP responses that can be returned
	// for specific paths instead of forwarding the request.
	HTTPRoutes []HTTPRoute
}

// ForwardTarget holds information how to forward to a target.
type ForwardTarget struct {
	// Port is the port to forward to.
	Port int32
	// Address is the address to forward to.
	Address string
}

// HTTPRoute defines a predefined HTTP response configuration for a specific path.
type HTTPRoute struct {
	// Location specifies the request path pattern to match for this response.
	Location string

	// Code is the HTTP status code to return for this response.
	Code int
	// Headers contains additional HTTP headers to include in the response.
	Headers []HTTPRouteHeader
	// Body contains the response body content to return.
	Body string
}

// HTTPRouteHeader defines a single HTTP header key-value pair.
type HTTPRouteHeader struct {
	// Name is the HTTP header name.
	Name string
	// Value is the HTTP header value.
	Value string
}

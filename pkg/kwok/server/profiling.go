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
	"net/http"
	"net/http/pprof"
	"runtime"
)

// InstallProfilingHandler registers the HTTP request patterns for /debug/pprof endpoint.
func (s *Server) InstallProfilingHandler(enableProfilingLogHandler bool, enableContentionProfiling bool) {
	if !enableProfilingLogHandler {
		s.restfulCont.Handle(pprofBasePath, getHandlerForDisabledEndpoint("profiling endpoint is disabled."))
		return
	}

	// Setup pprof handlers.
	s.restfulCont.Handle(pprofBasePath, http.HandlerFunc(pprof.Index))
	if enableContentionProfiling {
		runtime.SetBlockProfileRate(1)
	}
}

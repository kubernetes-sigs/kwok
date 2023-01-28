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

	"sigs.k8s.io/kwok/pkg/log"
)

func (s *Server) healthzCheck(rw http.ResponseWriter, req *http.Request) {
	_, err := rw.Write([]byte("ok"))
	if err != nil {
		logger := log.FromContext(req.Context())
		logger.Error("Failed to write", err)
	}
}

// InstallHealthz installs the healthz handler.
func (s *Server) InstallHealthz() {
	s.restfulCont.Handle("/healthz", http.HandlerFunc(s.healthzCheck))
	s.restfulCont.Handle("/readyz", http.HandlerFunc(s.healthzCheck))
	s.restfulCont.Handle("/livez", http.HandlerFunc(s.healthzCheck))
}

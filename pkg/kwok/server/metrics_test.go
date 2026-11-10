/*
Copyright 2026 The Kubernetes Authors.

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
	"strings"
	"testing"

	"sigs.k8s.io/kwok/pkg/log"
)

type disabledWorkloadContextDataSource struct{}

func (disabledWorkloadContextDataSource) ListPods(string) ([]log.ObjectRef, bool) {
	return nil, false
}

func (disabledWorkloadContextDataSource) ListNodes() []string {
	return nil
}

func (disabledWorkloadContextDataSource) StartedContainersTotal(string) int64 {
	return 0
}

func (disabledWorkloadContextDataSource) WorkloadContextEnabled() bool {
	return false
}

func TestInitCELDisablesUnavailableWorkloadContext(t *testing.T) {
	server := &Server{dataSource: disabledWorkloadContextDataSource{}}
	if err := server.initCEL(); err != nil {
		t.Fatal(err)
	}
	if _, err := server.env.Compile(`1.0`); err != nil {
		t.Fatalf("ordinary metric expression should compile: %v", err)
	}
	if _, err := server.env.Compile(`double(workload.replicas)`); err == nil ||
		!strings.Contains(err.Error(), "cluster-wide workload membership") {
		t.Fatalf("workload expression error = %v, want cluster-wide membership error", err)
	}
}

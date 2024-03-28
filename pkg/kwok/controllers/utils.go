/*
Copyright 2022 The Kubernetes Authors.

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

package controllers

import (
	"bytes"
	"encoding/json"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/strategicpatch"

	utilsnet "sigs.k8s.io/kwok/pkg/utils/net"
	"sigs.k8s.io/kwok/pkg/utils/wait"
)

func parseCIDR(s string) (*net.IPNet, error) {
	return utilsnet.ParseCIDR(s)
}

func addIP(ip net.IP, add uint64) net.IP {
	return utilsnet.AddIP(ip, add)
}

type ipPool struct {
	mut    sync.Mutex
	used   map[string]struct{}
	usable map[string]struct{}
	cidr   *net.IPNet
	index  uint64
}

func newIPPool(cidr *net.IPNet) *ipPool {
	return &ipPool{
		used:   make(map[string]struct{}),
		usable: make(map[string]struct{}),
		cidr:   cidr,
	}
}

func (i *ipPool) new() string {
	for {
		ip := addIP(i.cidr.IP, i.index).String()
		i.index++

		if _, ok := i.used[ip]; ok {
			continue
		}

		i.used[ip] = struct{}{}
		i.usable[ip] = struct{}{}
		return ip
	}
}

func (i *ipPool) Get() string {
	i.mut.Lock()
	defer i.mut.Unlock()
	ip := ""
	if len(i.usable) != 0 {
		for s := range i.usable {
			ip = s
			break
		}
	}
	if ip == "" {
		ip = i.new()
	}
	delete(i.usable, ip)
	i.used[ip] = struct{}{}
	return ip
}

func (i *ipPool) Put(ip string) {
	i.mut.Lock()
	defer i.mut.Unlock()
	if !i.cidr.Contains(net.ParseIP(ip)) {
		return
	}
	delete(i.used, ip)
	i.usable[ip] = struct{}{}
}

func (i *ipPool) Use(ip string) {
	i.mut.Lock()
	defer i.mut.Unlock()
	if !i.cidr.Contains(net.ParseIP(ip)) {
		return
	}
	i.used[ip] = struct{}{}
}

func labelsParse(selector string) (labels.Selector, error) {
	if selector == "" {
		return nil, nil
	}
	return labels.Parse(selector)
}

type resourceStageJob[T any] struct {
	Resource T
	Stage    *LifecycleStage
	Key      string
	// RetryCount is used for tracking the retry times of a job.
	// Must be initialized to 0.
	RetryCount *uint64
}

// defaultBackoff provides a backoff setting for kwok controllers to apply failed jobs
func defaultBackoff() wait.Backoff {
	return wait.Backoff{Duration: 1 * time.Second, Factor: 2.0, Jitter: 0.2, Cap: 32 * time.Minute}
}

// backoffDelayByStep calculates the backoff delay period based on steps
func backoffDelayByStep(steps uint64, c wait.Backoff) time.Duration {
	delay := math.Min(
		float64(c.Duration)*math.Pow(c.Factor, float64(steps)),
		float64(c.Cap))
	return wait.Jitter(time.Duration(delay), c.Jitter)
}

// shouldRetry determines if a certain error needs to be retried
func shouldRetry(err error) bool {
	// if apiserver is not reachable
	if utilnet.IsConnectionRefused(err) {
		return true
	}
	// if it is a network issue reported by apiserver side
	if apierrors.IsServerTimeout(err) ||
		apierrors.IsTimeout(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTooManyRequests(err) {
		return true
	}
	// ignore all other cases
	return false
}

// checkNeedPatch checks if the object needs to be patched
func checkNeedPatch[T any](obj T, patchData []byte) (bool, error) {
	original, err := json.Marshal(obj)
	if err != nil {
		return false, err
	}

	sum, err := strategicpatch.StrategicMergePatch(original, patchData, obj)
	if err != nil {
		return false, err
	}

	var tmp T
	err = json.Unmarshal(sum, &tmp)
	if err != nil {
		return false, err
	}

	dist, err := json.Marshal(tmp)
	if err != nil {
		return false, err
	}

	if bytes.Equal(original, dist) {
		return false, nil
	}

	return true, nil
}

// wrapPatchData wraps the patch data with the parent key
func wrapPatchData(parent string, patchData []byte) []byte {
	if parent == "" {
		return patchData
	}
	return []byte("{" + strconv.Quote(parent) + ":" + string(patchData) + "}")
}

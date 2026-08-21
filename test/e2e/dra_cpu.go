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

package e2e

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/test/e2e/helper"
)

const (
	// draCPUDriver is the driver name published by both the real
	// dra-driver-cpu and the kwok simulation in kustomize/stage/dra/cpu.
	draCPUDriver = "dra.cpu"

	// draCPUDeviceClass is the DeviceClass installed by the dra-driver-cpu
	// helm chart, selecting all devices of draCPUDriver. The kwok simulated
	// devices must be allocatable through the very same class.
	draCPUDeviceClass = "dra.cpu"

	// draCPUCapacity is the consumable capacity published per device in the
	// default grouped mode.
	draCPUCapacity = "dra.cpu/cpu"

	// draCPUAnnotation and draCPUNUMAAnnotation opt a fake node into the
	// cpu-resource-slice-publish stage, draCPUModeAnnotation selects the
	// device mode.
	draCPUAnnotation     = "kwok.x-k8s.io/dra-cpu"
	draCPUNUMAAnnotation = "kwok.x-k8s.io/dra-cpu-numa"
	draCPUModeAnnotation = "kwok.x-k8s.io/dra-cpu-mode"

	waitTimeout = 5 * time.Minute
)

// knownDivergentCPUAttributes are attributes whose values depend on the real
// hardware and are fixed constants in the simulation, so only their presence
// and type are compared.
var knownDivergentCPUAttributes = map[resourcev1.QualifiedName]string{
	// The simulation hardcodes smtEnabled=false while real hardware may
	// have SMT enabled.
	"dra.cpu/smtEnabled": "simulation hardcodes smtEnabled=false",
}

// knownAheadCPUAttributes are attributes the simulation publishes that the
// pinned driver release does not yet: the simulation tracks the driver's main
// branch. Remove entries when bumping draDriverCPUVersion past their release.
var knownAheadCPUAttributes = map[resourcev1.QualifiedName]string{
	// Standard NUMA attribute published by dra-driver-cpu main, not in v0.2.0.
	"resource.kubernetes.io/numaNode": "published by dra-driver-cpu main, not yet in v0.2.0",
}

// cpuDeviceContract lists the attributes the real dra-driver-cpu publishes
// per device mode with their types, mirroring the checks in its own e2e suite
// (test/e2e/resource_slice_test.go), so that ResourceClaims written for the
// real driver keep working against the simulated devices.
func cpuDeviceContract(mode string) map[resourcev1.QualifiedName]string {
	if mode == "individual" {
		return map[resourcev1.QualifiedName]string{
			"resource.kubernetes.io/numaNode": "int",
			"dra.cpu/socketID":                "int",
			"dra.cpu/smtEnabled":              "bool",
			"dra.cpu/cacheL3ID":               "int",
			"dra.cpu/coreType":                "string",
			"dra.cpu/coreID":                  "int",
			"dra.cpu/cpuID":                   "int",
		}
	}
	// grouped by NUMA node, the driver default
	return map[resourcev1.QualifiedName]string{
		"resource.kubernetes.io/numaNode": "int",
		"dra.cpu/socketID":                "int",
		"dra.cpu/smtEnabled":              "bool",
		"dra.cpu/numCPUs":                 "int",
	}
}

// checkCPUDeviceContract verifies the devices against the per-mode attribute
// contract of the real driver. Attributes in optional may be absent (version
// skew between the pinned release and the driver's main branch).
func checkCPUDeviceContract(t *testing.T, what, mode string, devices []resourcev1.Device, optional map[resourcev1.QualifiedName]string) {
	t.Helper()

	contract := cpuDeviceContract(mode)
	for _, d := range devices {
		for name, typ := range contract {
			attr, ok := d.Attributes[name]
			if !ok {
				if reason, skewed := optional[name]; skewed {
					t.Logf("%s: device %q attribute %q absent (%s)", what, d.Name, name, reason)
				} else {
					t.Errorf("%s: device %q is missing attribute %q required by the dra-driver-cpu contract", what, d.Name, name)
				}
				continue
			}
			if attributeType(attr) != typ {
				t.Errorf("%s: device %q attribute %q has type %s, want %s", what, d.Name, name, attributeType(attr), typ)
			}
		}
		if mode != "individual" {
			if _, ok := d.Capacity[draCPUCapacity]; !ok {
				t.Errorf("%s: device %q is missing the consumable capacity %q", what, d.Name, draCPUCapacity)
			}
			if d.AllowMultipleAllocations == nil || !*d.AllowMultipleAllocations {
				t.Errorf("%s: device %q must allow multiple allocations", what, d.Name)
			}
		}
	}
}

type draCPUContextKey string

const realCPUSlicesKey draCPUContextKey = "realCPUSlices"

// realCPUSliceInfo captures what the real driver published, used to derive an
// equivalent fake node and to compare against the simulated slice.
type realCPUSliceInfo struct {
	NodeName  string
	Devices   []resourcev1.Device
	TotalCPUs int64
}

// listRealCPUSlices returns the ResourceSlices published by the real
// dra-driver-cpu, excluding any slice belonging to the given fake node.
func listRealCPUSlices(ctx context.Context, client *resources.Resources, kwokNodeName string) ([]resourcev1.ResourceSlice, error) {
	slices := &resourcev1.ResourceSliceList{}
	if err := client.List(ctx, slices); err != nil {
		return nil, err
	}
	var out []resourcev1.ResourceSlice
	for _, s := range slices.Items {
		if s.Spec.Driver != draCPUDriver {
			continue
		}
		if s.Spec.NodeName == nil || *s.Spec.NodeName == kwokNodeName {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

// attributeValueString renders a DeviceAttribute for error messages and
// typed comparison.
func attributeValueString(a resourcev1.DeviceAttribute) string {
	switch {
	case a.IntValue != nil:
		return fmt.Sprintf("int:%d", *a.IntValue)
	case a.BoolValue != nil:
		return fmt.Sprintf("bool:%t", *a.BoolValue)
	case a.StringValue != nil:
		return fmt.Sprintf("string:%q", *a.StringValue)
	case a.VersionValue != nil:
		return fmt.Sprintf("version:%q", *a.VersionValue)
	default:
		return "<empty>"
	}
}

func attributeType(a resourcev1.DeviceAttribute) string {
	switch {
	case a.IntValue != nil:
		return "int"
	case a.BoolValue != nil:
		return "bool"
	case a.StringValue != nil:
		return "string"
	case a.VersionValue != nil:
		return "version"
	default:
		return "<empty>"
	}
}

// compareCPUDevices compares the devices published by the real driver with
// the simulated ones and reports any divergence that is not an allowlisted,
// documented gap.
func compareCPUDevices(t *testing.T, real, sim []resourcev1.Device) {
	t.Helper()

	if len(real) != len(sim) {
		t.Fatalf("device count mismatch: real driver published %d devices, simulation published %d", len(real), len(sim))
	}

	// Sort copies: the callers' slices alias data kept in the feature context.
	real = slices.Clone(real)
	sim = slices.Clone(sim)
	sortDevices := func(devices []resourcev1.Device) {
		sort.Slice(devices, func(i, j int) bool { return devices[i].Name < devices[j].Name })
	}
	sortDevices(real)
	sortDevices(sim)

	for i := range real {
		rd, sd := real[i], sim[i]

		if rd.Name != sd.Name {
			t.Errorf("device %d name mismatch: real=%q sim=%q", i, rd.Name, sd.Name)
			continue
		}

		rMulti := rd.AllowMultipleAllocations != nil && *rd.AllowMultipleAllocations
		sMulti := sd.AllowMultipleAllocations != nil && *sd.AllowMultipleAllocations
		if rMulti != sMulti {
			t.Errorf("device %q allowMultipleAllocations mismatch: real=%t sim=%t", rd.Name, rMulti, sMulti)
		}

		// Capacity must match exactly: it drives consumable-capacity
		// allocation, the core of the grouped mode.
		for name, rc := range rd.Capacity {
			sc, ok := sd.Capacity[name]
			if !ok {
				t.Errorf("device %q capacity %q published by real driver is missing in simulation", rd.Name, name)
				continue
			}
			if rc.Value.Cmp(sc.Value) != 0 {
				t.Errorf("device %q capacity %q value mismatch: real=%s sim=%s", rd.Name, name, rc.Value.String(), sc.Value.String())
			}
		}
		for name := range sd.Capacity {
			if _, ok := rd.Capacity[name]; !ok {
				t.Errorf("device %q capacity %q published by simulation does not exist on the real driver", rd.Name, name)
			}
		}

		// Attributes: every real attribute must exist in the simulation with
		// the same type and value, unless allowlisted above.
		for name, ra := range rd.Attributes {
			sa, ok := sd.Attributes[name]
			if !ok {
				t.Errorf("device %q attribute %q published by real driver is missing in simulation", rd.Name, name)
				continue
			}
			if attributeType(ra) != attributeType(sa) {
				t.Errorf("device %q attribute %q type mismatch: real=%s sim=%s", rd.Name, name, attributeValueString(ra), attributeValueString(sa))
				continue
			}
			if attributeValueString(ra) != attributeValueString(sa) {
				if reason, known := knownDivergentCPUAttributes[name]; known {
					t.Logf("known gap: device %q attribute %q value differs: real=%s sim=%s (%s)", rd.Name, name, attributeValueString(ra), attributeValueString(sa), reason)
				} else {
					t.Errorf("device %q attribute %q value mismatch: real=%s sim=%s", rd.Name, name, attributeValueString(ra), attributeValueString(sa))
				}
			}
		}
		for name := range sd.Attributes {
			if _, ok := rd.Attributes[name]; !ok {
				if reason, skewed := knownAheadCPUAttributes[name]; skewed {
					t.Logf("known skew: device %q attribute %q only in simulation (%s)", rd.Name, name, reason)
				} else {
					t.Errorf("device %q attribute %q published by simulation does not exist on the real driver", rd.Name, name)
				}
			}
		}
	}
}

// newCPUClaim returns a ResourceClaim requesting one CPU of consumable
// capacity through the DeviceClass installed by the real driver, the same
// shape a user of dra-driver-cpu would write.
func newCPUClaim(name, namespace string) *resourcev1.ResourceClaim {
	return newCPUClaimFromRequests(name, namespace, capacityRequest("cpus", 1))
}

// capacityRequest returns a device request for the given amount of consumable
// CPU capacity, mirroring dra-driver-cpu's makeResourceClaimSpec in grouped
// mode.
func capacityRequest(name string, cpus int64) resourcev1.DeviceRequest {
	return resourcev1.DeviceRequest{
		Name: name,
		Exactly: &resourcev1.ExactDeviceRequest{
			DeviceClassName: draCPUDeviceClass,
			Capacity: &resourcev1.CapacityRequirements{
				Requests: map[resourcev1.QualifiedName]resource.Quantity{
					draCPUCapacity: *resource.NewQuantity(cpus, resource.DecimalSI),
				},
			},
		},
	}
}

func newCPUClaimFromRequests(name, namespace string, requests ...resourcev1.DeviceRequest) *resourcev1.ResourceClaim {
	return &resourcev1.ResourceClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimSpec{
			Devices: resourcev1.DeviceClaim{
				Requests: requests,
			},
		},
	}
}

// countRequest returns a device request for the given number of exclusive
// devices, restricted to the per-CPU devices of the individual mode via
// their cpuID attribute.
func countRequest(count int64) resourcev1.DeviceRequest {
	return resourcev1.DeviceRequest{
		Name: "cpus",
		Exactly: &resourcev1.ExactDeviceRequest{
			DeviceClassName: draCPUDeviceClass,
			Count:           count,
			Selectors: []resourcev1.DeviceSelector{
				{CEL: &resourcev1.CELDeviceSelector{Expression: `has(device.attributes["dra.cpu"].cpuID)`}},
			},
		},
	}
}

// newCPUClaimTemplate returns a ResourceClaimTemplate whose generated claims
// match newCPUClaim, the consumption shape used throughout the real driver's
// e2e suite.
func newCPUClaimTemplate(name, namespace string) *resourcev1.ResourceClaimTemplate {
	return &resourcev1.ResourceClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: resourcev1.ResourceClaimTemplateSpec{
			Spec: newCPUClaim(name, namespace).Spec,
		},
	}
}

// deviceByName looks a device up in a published device list.
func deviceByName(devices []resourcev1.Device, name string) *resourcev1.Device {
	for i := range devices {
		if devices[i].Name == name {
			return &devices[i]
		}
	}
	return nil
}

// newCPUClaimPod returns a Pod referencing the given claim with the mirrored
// CPU request required by dra-driver-cpu's workload requirements.
func newCPUClaimPod(name, namespace, image string, claim *resourcev1.ResourceClaim, nodeSelector map[string]string) *corev1.Pod {
	return newCPUClaimPodWithCPUs(name, namespace, image, claim, nodeSelector, 1)
}

func newCPUClaimPodWithCPUs(name, namespace, image string, claim *resourcev1.ResourceClaim, nodeSelector map[string]string, cpus int64) *corev1.Pod {
	pod := helper.NewPodBuilder(name).
		WithNamespace(namespace).
		Build()
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = []corev1.Container{
		{
			Name:  "app",
			Image: image,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(cpus, resource.DecimalSI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(cpus, resource.DecimalSI),
				},
				Claims: []corev1.ResourceClaim{
					{Name: "cpus"},
				},
			},
		},
	}
	pod.Spec.NodeSelector = nodeSelector
	pod.Spec.ResourceClaims = []corev1.PodResourceClaim{
		{
			Name:              "cpus",
			ResourceClaimName: &claim.Name,
		},
	}
	return pod
}

// newCPUClaimTemplatePod returns a Pod shaped like newCPUClaimPod but
// generating its claim from the given ResourceClaimTemplate.
func newCPUClaimTemplatePod(name, namespace, image, templateName string, nodeSelector map[string]string) *corev1.Pod {
	pod := helper.NewPodBuilder(name).
		WithNamespace(namespace).
		Build()
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = []corev1.Container{
		{
			Name:  "app",
			Image: image,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(1, resource.DecimalSI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(1, resource.DecimalSI),
				},
				Claims: []corev1.ResourceClaim{
					{Name: "cpus"},
				},
			},
		},
	}
	pod.Spec.NodeSelector = nodeSelector
	pod.Spec.ResourceClaims = []corev1.PodResourceClaim{
		{
			Name:                      "cpus",
			ResourceClaimTemplateName: &templateName,
		},
	}
	return pod
}

// newMultiCPUClaimPod returns a Pod with one container per claim, each
// mirroring its own claim's CPUs, the multi-container shape of the driver's
// individual mode example.
func newMultiCPUClaimPod(name, namespace, image string, nodeSelector map[string]string, claims []*resourcev1.ResourceClaim, cpus []int64) *corev1.Pod {
	pod := helper.NewPodBuilder(name).
		WithNamespace(namespace).
		Build()
	pod.Spec.InitContainers = nil
	pod.Spec.Containers = nil
	pod.Spec.NodeSelector = nodeSelector
	pod.Spec.ResourceClaims = nil
	for i, claim := range claims {
		ref := fmt.Sprintf("cpus-%d", i)
		pod.Spec.Containers = append(pod.Spec.Containers, corev1.Container{
			Name:  fmt.Sprintf("app-%d", i),
			Image: image,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(cpus[i], resource.DecimalSI),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: *resource.NewQuantity(cpus[i], resource.DecimalSI),
				},
				Claims: []corev1.ResourceClaim{
					{Name: ref},
				},
			},
		})
		pod.Spec.ResourceClaims = append(pod.Spec.ResourceClaims, corev1.PodResourceClaim{
			Name:              ref,
			ResourceClaimName: &claim.Name,
		})
	}
	return pod
}

// runCPUClaimLifecycle creates the claim and pod, waits for the pod to run
// and the claim to be allocated and reserved, and returns the refreshed claim.
func runCPUClaimLifecycle(ctx context.Context, t *testing.T, client *resources.Resources, claim *resourcev1.ResourceClaim, pod *corev1.Pod) *resourcev1.ResourceClaim {
	t.Helper()

	t.Log("creating resource claim", log.KObj(claim))
	if err := client.Create(ctx, claim); err != nil {
		t.Fatal(err)
	}
	t.Log("creating pod with resource claim", log.KObj(pod))
	if err := client.Create(ctx, pod); err != nil {
		t.Fatal(err)
	}

	t.Log("waiting for pod to be running", log.KObj(pod))
	err := wait.For(
		conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
		wait.WithContext(ctx),
		wait.WithTimeout(waitTimeout),
	)
	if err != nil {
		t.Fatal("pod did not reach Running phase:", err)
	}

	if err := client.Get(ctx, claim.Name, claim.Namespace, claim); err != nil {
		t.Fatal(err)
	}
	if claim.Status.Allocation == nil {
		t.Fatal("resource claim is not allocated")
	}
	results := claim.Status.Allocation.Devices.Results
	if len(results) == 0 {
		t.Fatal("resource claim has no allocation results")
	}
	for _, r := range results {
		if r.Driver != draCPUDriver {
			t.Fatalf("unexpected allocation results: %+v", results)
		}
	}
	if len(claim.Status.ReservedFor) != 1 || claim.Status.ReservedFor[0].UID != pod.UID {
		t.Fatalf("resource claim is not reserved for the pod: %+v", claim.Status.ReservedFor)
	}
	t.Log("resource claim is allocated and reserved", log.KObj(claim))
	return claim
}

// waitPodUnschedulable waits until the scheduler reports the pod as
// unschedulable, the positive signal that scheduling is blocked rather than
// merely not having happened yet.
func waitPodUnschedulable(ctx context.Context, t *testing.T, client *resources.Resources, pod *corev1.Pod) {
	t.Helper()

	t.Log("waiting for pod to be reported unschedulable", log.KObj(pod))
	err := wait.For(
		conditions.New(client).ResourceMatch(pod, func(obj k8s.Object) bool {
			for _, cond := range obj.(*corev1.Pod).Status.Conditions {
				if cond.Type == corev1.PodScheduled && cond.Status == corev1.ConditionFalse && cond.Reason == corev1.PodReasonUnschedulable {
					return true
				}
			}
			return false
		}),
		wait.WithContext(ctx),
		wait.WithTimeout(waitTimeout),
	)
	if err != nil {
		t.Fatal("pod was not reported unschedulable:", err)
	}
}

// deleteClaimPod removes a claim and its pod, waiting for both to be gone so
// the consumed capacity is released for the next case.
func deleteClaimPod(ctx context.Context, t *testing.T, client *resources.Resources, claim *resourcev1.ResourceClaim, pod *corev1.Pod) {
	t.Helper()
	if err := client.Delete(ctx, pod); err != nil {
		t.Fatal(err)
	}
	err := wait.For(
		conditions.New(client).ResourceDeleted(pod),
		wait.WithContext(ctx),
		wait.WithTimeout(waitTimeout),
	)
	if err != nil {
		t.Fatal("pod was not deleted:", err)
	}
	if err := client.Delete(ctx, claim); err != nil {
		t.Fatal(err)
	}
	err = wait.For(
		conditions.New(client).ResourceDeleted(claim),
		wait.WithContext(ctx),
		wait.WithTimeout(waitTimeout),
	)
	if err != nil {
		t.Fatal("claim was not deleted:", err)
	}
}

// compareCPUAllocations compares the allocation the real driver stack
// produced with the one produced against the simulated slice.
func compareCPUAllocations(t *testing.T, real, sim *resourcev1.DeviceRequestAllocationResult) {
	t.Helper()

	if real.Driver != sim.Driver {
		t.Errorf("allocation driver mismatch: real=%q sim=%q", real.Driver, sim.Driver)
	}
	if real.Device != sim.Device {
		t.Errorf("allocation device mismatch: real=%q sim=%q", real.Device, sim.Device)
	}
	if (real.ShareID != nil) != (sim.ShareID != nil) {
		t.Errorf("allocation shareID presence mismatch: real=%v sim=%v", real.ShareID, sim.ShareID)
	}
	for name, rc := range real.ConsumedCapacity {
		sc, ok := sim.ConsumedCapacity[name]
		if !ok {
			t.Errorf("allocation consumed capacity %q missing in simulation", name)
			continue
		}
		if rc.Cmp(sc) != 0 {
			t.Errorf("allocation consumed capacity %q mismatch: real=%s sim=%s", name, rc.String(), sc.String())
		}
	}
	for name := range sim.ConsumedCapacity {
		if _, ok := real.ConsumedCapacity[name]; !ok {
			t.Errorf("allocation consumed capacity %q reported by simulation does not exist on the real allocation", name)
		}
	}
}

// dumpDRACPUDiagnostics prints the state needed to diagnose a missing
// simulated ResourceSlice: the Stage objects, the fake node, all
// ResourceSlices, and the kwok controller logs.
func dumpDRACPUDiagnostics(ctx context.Context, t *testing.T, cfg *envconf.Config, kwokNodeName string) {
	t.Helper()

	client, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		t.Log("diagnostics: failed to create client:", err)
		return
	}

	stages := &unstructured.UnstructuredList{}
	stages.SetGroupVersionKind(schema.GroupVersionKind{Group: "kwok.x-k8s.io", Version: "v1alpha1", Kind: "StageList"})
	if err := client.List(ctx, stages); err != nil {
		t.Log("diagnostics: failed to list stages:", err)
	} else {
		names := make([]string, 0, len(stages.Items))
		for _, s := range stages.Items {
			names = append(names, s.GetName())
		}
		t.Log("diagnostics: stages:", names)
	}

	node := &corev1.Node{}
	if err := client.Get(ctx, kwokNodeName, "", node); err != nil {
		t.Log("diagnostics: failed to get fake node:", err)
	} else {
		t.Logf("diagnostics: fake node annotations: %v", node.Annotations)
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				t.Logf("diagnostics: fake node Ready condition: %s", cond.Status)
			}
		}
	}

	slices := &resourcev1.ResourceSliceList{}
	if err := client.List(ctx, slices); err != nil {
		t.Log("diagnostics: failed to list resource slices:", err)
	} else {
		for _, s := range slices.Items {
			nodeName := ""
			if s.Spec.NodeName != nil {
				nodeName = *s.Spec.NodeName
			}
			t.Logf("diagnostics: resource slice %q driver=%q node=%q devices=%d", s.Name, s.Spec.Driver, nodeName, len(s.Spec.Devices))
		}
	}

	clientset, err := kubernetes.NewForConfig(cfg.Client().RESTConfig())
	if err != nil {
		t.Log("diagnostics: failed to create clientset:", err)
		return
	}
	pods, err := clientset.CoreV1().Pods("kube-system").List(ctx, metav1.ListOptions{LabelSelector: "app=kwok-controller"})
	if err != nil || len(pods.Items) == 0 {
		t.Log("diagnostics: failed to find kwok-controller pod:", err)
		return
	}
	for _, pod := range pods.Items {
		tailLines := int64(100)
		data, err := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{TailLines: &tailLines}).DoRaw(ctx)
		if err != nil {
			t.Logf("diagnostics: failed to get logs of %s: %v", pod.Name, err)
			continue
		}
		t.Logf("diagnostics: kwok-controller logs (%s):\n%s", pod.Name, string(data))
	}
}

// CaseDRACPU exercises the full DRA CPU lifecycle against the real
// dra-driver-cpu (installed by the suite setup) and the kwok simulation in
// kustomize/stage/dra/cpu, comparing the two at every step:
//
//  1. The real driver publishes ResourceSlices for the real node.
//  2. A fake node annotated with the observed topology is created and the
//     simulation publishes an equivalent ResourceSlice, which is compared
//     device by device against the real one (skipped on asymmetric-NUMA
//     hosts); both sides must satisfy the attribute contract asserted by
//     the driver's own e2e suite.
//  3. The same claim and pod shape is run against both nodes; both pods reach
//     Running and both claims are allocated and reserved, and the allocation
//     results are compared.
//  4. A second claim and pod per side shares the consumable capacity of the
//     same device with a distinct shareID.
//  5. Deleting the pods releases the reservations.
//  6. Claim shapes from the real driver's e2e suite and user documentation
//     (CEL topology selectors, matchAttribute over the cross-driver NUMA
//     attribute) allocate identically on both sides.
//  7. A claim splitting CPUs across NUMA nodes per the driver README
//     allocates from both NUMA nodes of a dedicated two-NUMA fake node.
//  8. A fake node with a fixed topology publishes in individual device mode;
//     a multi-container pod consuming one claim per container allocates
//     distinct exclusive devices.
//  9. ResourceClaimTemplates generate per-pod claims on both sides that
//     allocate identically and are garbage collected with their pods.
//  10. A claim with an unsatisfiable CEL selector leaves the pod pending on
//     both sides without an allocation.
//  11. A pod contending for the exhausted capacity of a capacity-bounded fake
//     node stays pending until the blocking claim is released.
//  12. Two pods share one claim on the simulation and are both reserved; the
//     real driver would reject the second pod at the node level, but kwok
//     has no node agent, so only scheduler-level semantics are verified.
//  13. Exhausting all exclusive individual devices leaves a pod pending until
//     the blocking claim is released.
//  14. Deleting the fake nodes garbage collects the simulated ResourceSlices.
func CaseDRACPU(kwokNodeName, namespace string) *features.FeatureBuilder {
	node := helper.NewNodeBuilder(kwokNodeName).Build()
	individualNode := helper.NewNodeBuilder(kwokNodeName + "-ind").Build()
	numaNode := helper.NewNodeBuilder(kwokNodeName + "-numa").Build()
	// A grouped mode node with a single 4-CPU device, for the capacity
	// exhaustion and claim sharing assessments.
	capNode := helper.NewNodeBuilder(kwokNodeName+"-cap").
		WithAnnotation(draCPUAnnotation, "4").
		WithAnnotation(draCPUNUMAAnnotation, "1").
		Build()
	// The builder only labels fake nodes type:kwok, which is ambiguous once
	// several fake nodes coexist; pods pin a specific node by hostname.
	for _, n := range []*corev1.Node{node, individualNode, numaNode, capNode} {
		n.Labels["kubernetes.io/hostname"] = n.Name
	}

	simSliceName := kwokNodeName + "-" + draCPUDriver
	individualSliceName := individualNode.Name + "-" + draCPUDriver
	numaSliceName := numaNode.Name + "-" + draCPUDriver
	capSliceName := capNode.Name + "-" + draCPUDriver

	realClaim := newCPUClaim("real-cpu-claim", namespace)
	kwokClaim := newCPUClaim("kwok-cpu-claim", namespace)
	realShareClaim := newCPUClaim("real-cpu-claim-share", namespace)
	kwokShareClaim := newCPUClaim("kwok-cpu-claim-share", namespace)

	// The real pods run pause containers pinned by the real driver; the
	// node selector is filled in once the real node is known.
	realPod := newCPUClaimPod("real-cpu-pod", namespace, "registry.k8s.io/pause:3.10", realClaim, nil)
	kwokPod := newCPUClaimPod("kwok-cpu-pod", namespace, "fake-image", kwokClaim, map[string]string{"type": "kwok"})
	realSharePod := newCPUClaimPod("real-cpu-pod-share", namespace, "registry.k8s.io/pause:3.10", realShareClaim, nil)
	kwokSharePod := newCPUClaimPod("kwok-cpu-pod-share", namespace, "fake-image", kwokShareClaim, map[string]string{"type": "kwok"})

	return features.New("DRA CPU: real dra-driver-cpu vs kwok simulation").
		// ── real driver → ResourceSlice ────────────────────────────────────
		Assess("real driver publishes resource slices", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			t.Log("waiting for the real dra-driver-cpu to publish resource slices")
			var real []resourcev1.ResourceSlice
			err = wait.For(
				func(ctx context.Context) (bool, error) {
					real, err = listRealCPUSlices(ctx, client, kwokNodeName)
					if err != nil {
						return false, err
					}
					return len(real) > 0, nil
				},
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				// The driver enumerates CPUs from the node's sysfs and
				// publishes nothing when per-CPU NUMA topology is absent
				// (e.g. Docker Desktop VMs).
				t.Fatal("real driver did not publish any resource slice (the node must expose NUMA topology in sysfs):", err)
			}

			info := realCPUSliceInfo{NodeName: *real[0].Spec.NodeName}
			for _, s := range real {
				if *s.Spec.NodeName != info.NodeName {
					t.Fatalf("expected slices for a single real node, got %q and %q", info.NodeName, *s.Spec.NodeName)
				}
				info.Devices = append(info.Devices, s.Spec.Devices...)
			}
			for _, d := range info.Devices {
				capacity, ok := d.Capacity[draCPUCapacity]
				if !ok {
					t.Fatalf("real device %q has no %q capacity; expected the driver default grouped mode", d.Name, draCPUCapacity)
				}
				info.TotalCPUs += capacity.Value.Value()
			}
			t.Logf("real driver published %d devices with %d CPUs on node %q", len(info.Devices), info.TotalCPUs, info.NodeName)

			return context.WithValue(ctx, realCPUSlicesKey, info)
		}).
		// ── fake node → simulated ResourceSlice, compared with the real one ─
		Assess("simulation publishes an equivalent resource slice", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			info, ok := ctx.Value(realCPUSlicesKey).(realCPUSliceInfo)
			if !ok {
				t.Fatal("requires the 'real driver publishes resource slices' assessment to have run")
			}

			// Mirror the observed topology onto the fake node.
			node.Annotations[draCPUAnnotation] = strconv.FormatInt(info.TotalCPUs, 10)
			node.Annotations[draCPUNUMAAnnotation] = strconv.Itoa(len(info.Devices))
			ctx = helper.CreateNode(node)(ctx, t, cfg)

			t.Log("waiting for the simulated resource slice", simSliceName)
			simSlice := &resourcev1.ResourceSlice{}
			simSlice.Name = simSliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(simSlice, func(obj k8s.Object) bool {
					return obj.(*resourcev1.ResourceSlice).Spec.Driver == draCPUDriver
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				dumpDRACPUDiagnostics(ctx, t, cfg, kwokNodeName)
				t.Fatal("simulated resource slice was not published:", err)
			}
			if len(simSlice.Spec.Devices) != len(info.Devices) {
				t.Fatalf("simulation published %d devices, want %d", len(simSlice.Spec.Devices), len(info.Devices))
			}

			// The simulation splits the CPUs floor+remainder across the NUMA
			// groups in device-name order; on asymmetric-NUMA hosts the real
			// distribution differs, so compare device by device only when the
			// per-device capacities match in that order.
			n := int64(len(info.Devices))
			base, rem := info.TotalCPUs/n, info.TotalCPUs%n
			want := make([]int64, 0, n)
			for i := range n {
				if i < rem {
					want = append(want, base+1)
				} else {
					want = append(want, base)
				}
			}
			realDevices := slices.Clone(info.Devices)
			sort.Slice(realDevices, func(i, j int) bool { return realDevices[i].Name < realDevices[j].Name })
			got := make([]int64, 0, n)
			for _, d := range realDevices {
				capacity := d.Capacity[draCPUCapacity]
				got = append(got, capacity.Value.Value())
			}
			if !slices.Equal(want, got) {
				t.Logf("real driver CPU distribution %v differs from the even split %v (asymmetric-NUMA host); skipping the device-by-device comparison", got, want)
			} else {
				compareCPUDevices(t, info.Devices, simSlice.Spec.Devices)
			}
			checkCPUDeviceContract(t, "real driver", "grouped", info.Devices, knownAheadCPUAttributes)
			checkCPUDeviceContract(t, "simulation", "grouped", simSlice.Spec.Devices, nil)
			return ctx
		}).
		// ── the same claim and pod lifecycle on both nodes ─────────────────
		Assess("claim and pod lifecycle works against both and allocations match", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			info, ok := ctx.Value(realCPUSlicesKey).(realCPUSliceInfo)
			if !ok {
				t.Fatal("requires the 'real driver publishes resource slices' assessment to have run")
			}

			realPod.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": info.NodeName}
			realAllocated := runCPUClaimLifecycle(ctx, t, client, realClaim, realPod)
			kwokAllocated := runCPUClaimLifecycle(ctx, t, client, kwokClaim, kwokPod)

			realResult := realAllocated.Status.Allocation.Devices.Results[0]
			kwokResult := kwokAllocated.Status.Allocation.Devices.Results[0]
			if realResult.Pool != info.NodeName {
				t.Errorf("real allocation pool mismatch: got %q, want %q", realResult.Pool, info.NodeName)
			}
			if kwokResult.Pool != kwokNodeName {
				t.Errorf("kwok allocation pool mismatch: got %q, want %q", kwokResult.Pool, kwokNodeName)
			}
			compareCPUAllocations(t, &realResult, &kwokResult)
			return ctx
		}).
		// ── consumable capacity sharing ────────────────────────────────
		Assess("a second claim shares the device capacity with a distinct shareID", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			info, ok := ctx.Value(realCPUSlicesKey).(realCPUSliceInfo)
			if !ok {
				t.Fatal("requires the 'real driver publishes resource slices' assessment to have run")
			}

			realSharePod.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": info.NodeName}
			realShared := runCPUClaimLifecycle(ctx, t, client, realShareClaim, realSharePod)
			kwokShared := runCPUClaimLifecycle(ctx, t, client, kwokShareClaim, kwokSharePod)

			checkSharedAllocation := func(what string, first, second *resourcev1.ResourceClaim) {
				r1 := first.Status.Allocation.Devices.Results[0]
				r2 := second.Status.Allocation.Devices.Results[0]
				if r1.Device != r2.Device {
					t.Errorf("%s: expected both claims on the shared device, got %q and %q", what, r1.Device, r2.Device)
				}
				if r1.ShareID == nil || r2.ShareID == nil {
					t.Errorf("%s: both allocations must carry a shareID, got %v and %v", what, r1.ShareID, r2.ShareID)
				} else if *r1.ShareID == *r2.ShareID {
					t.Errorf("%s: shareIDs must be distinct, both are %q", what, *r1.ShareID)
				}
			}
			// Refresh the first claims to compare against.
			if err := client.Get(ctx, realClaim.Name, realClaim.Namespace, realClaim); err != nil {
				t.Fatal(err)
			}
			if err := client.Get(ctx, kwokClaim.Name, kwokClaim.Namespace, kwokClaim); err != nil {
				t.Fatal(err)
			}
			checkSharedAllocation("real driver", realClaim, realShared)
			checkSharedAllocation("simulation", kwokClaim, kwokShared)

			realResult := realShared.Status.Allocation.Devices.Results[0]
			kwokResult := kwokShared.Status.Allocation.Devices.Results[0]
			compareCPUAllocations(t, &realResult, &kwokResult)
			return ctx
		}).
		// ── pod deletion → reservations released ─────────────────────────
		Assess("pod deletion releases the claim reservations", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			for _, pod := range []*corev1.Pod{realPod, kwokPod, realSharePod, kwokSharePod} {
				t.Log("deleting pod", log.KObj(pod))
				if err = client.Delete(ctx, pod); err != nil {
					t.Fatal(err)
				}
				err = wait.For(
					conditions.New(client).ResourceDeleted(pod),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("pod was not deleted:", err)
				}
			}

			for _, claim := range []*resourcev1.ResourceClaim{realClaim, kwokClaim, realShareClaim, kwokShareClaim} {
				t.Log("waiting for claim reservation to be released", log.KObj(claim))
				err = wait.For(
					conditions.New(client).ResourceMatch(claim, func(obj k8s.Object) bool {
						c := obj.(*resourcev1.ResourceClaim)
						return len(c.Status.ReservedFor) == 0
					}),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("claim reservation was not released:", err)
				}
				if err = client.Delete(ctx, claim); err != nil {
					t.Fatal(err)
				}
				// Allocated claims hold device capacity until actually deleted;
				// the next assessments contend for the same capacity.
				err = wait.For(
					conditions.New(client).ResourceDeleted(claim),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("claim was not deleted:", err)
				}
			}
			return ctx
		}).
		// ── claim shapes from the real driver's e2e suite ──────────────────
		Assess("claims written for the real driver allocate identically", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			info, ok := ctx.Value(realCPUSlicesKey).(realCPUSliceInfo)
			if !ok {
				t.Fatal("requires the 'real driver publishes resource slices' assessment to have run")
			}

			simSlice := &resourcev1.ResourceSlice{}
			if err := client.Get(ctx, simSliceName, "", simSlice); err != nil {
				t.Fatal(err)
			}

			realSelector := map[string]string{"kubernetes.io/hostname": info.NodeName}
			kwokSelector := map[string]string{"type": "kwok"}

			runBoth := func(caseName string, cpus int64, requests []resourcev1.DeviceRequest, constraints []resourcev1.DeviceConstraint, check func(what string, devices []resourcev1.Device, results []resourcev1.DeviceRequestAllocationResult)) {
				for _, side := range []struct {
					what     string
					image    string
					selector map[string]string
					devices  []resourcev1.Device
				}{
					{"real driver", "registry.k8s.io/pause:3.10", realSelector, info.Devices},
					{"simulation", "fake-image", kwokSelector, simSlice.Spec.Devices},
				} {
					name := caseName + "-" + strings.Fields(side.what)[0]
					claim := newCPUClaimFromRequests(name, namespace, requests...)
					claim.Spec.Devices.Constraints = constraints
					pod := newCPUClaimPodWithCPUs(name, namespace, side.image, claim, side.selector, cpus)

					t.Logf("%s: running claim shape %q", side.what, caseName)
					allocated := runCPUClaimLifecycle(ctx, t, client, claim, pod)
					check(side.what, side.devices, allocated.Status.Allocation.Devices.Results)
					deleteClaimPod(ctx, t, client, claim, pod)
				}
			}

			// CEL selector over the driver's topology attributes, from the
			// device-attributes user documentation.
			celRequest := capacityRequest("cpus", 1)
			celRequest.Exactly.Selectors = []resourcev1.DeviceSelector{
				{CEL: &resourcev1.CELDeviceSelector{Expression: `device.attributes["dra.cpu"].numaNodeID == 0`}},
			}
			runBoth("cel-selector", 1,
				[]resourcev1.DeviceRequest{celRequest},
				nil,
				func(what string, devices []resourcev1.Device, results []resourcev1.DeviceRequestAllocationResult) {
					dev := deviceByName(devices, results[0].Device)
					if dev == nil {
						t.Errorf("%s: cel-selector: allocated device %q not found in published devices", what, results[0].Device)
						return
					}
					if numa := dev.Attributes["dra.cpu/numaNodeID"]; numa.IntValue == nil || *numa.IntValue != 0 {
						t.Errorf("%s: cel-selector: allocated device %q is not on NUMA node 0", what, results[0].Device)
					}
				},
			)

			// matchAttribute over the cross-driver NUMA attribute aligns both
			// requests, the alignment shape from the README. dra.net/numaNode is
			// used until the pinned release publishes the standard attribute.
			numaAttribute := resourcev1.FullyQualifiedName("dra.net/numaNode")
			runBoth("match-attribute", 2,
				[]resourcev1.DeviceRequest{capacityRequest("cpus-a", 1), capacityRequest("cpus-b", 1)},
				[]resourcev1.DeviceConstraint{{Requests: []string{"cpus-a", "cpus-b"}, MatchAttribute: &numaAttribute}},
				func(what string, devices []resourcev1.Device, results []resourcev1.DeviceRequestAllocationResult) {
					if len(results) != 2 {
						t.Errorf("%s: match-attribute: expected 2 allocation results, got %d", what, len(results))
						return
					}
					var numas []int64
					for _, r := range results {
						dev := deviceByName(devices, r.Device)
						if dev == nil {
							t.Errorf("%s: match-attribute: allocated device %q not found in published devices", what, r.Device)
							return
						}
						numa := dev.Attributes["dra.net/numaNode"]
						if numa.IntValue == nil {
							t.Errorf("%s: match-attribute: device %q has no dra.net/numaNode attribute", what, r.Device)
							return
						}
						numas = append(numas, *numa.IntValue)
					}
					if numas[0] != numas[1] {
						t.Errorf("%s: match-attribute: devices are on different NUMA nodes: %v", what, numas)
					}
				},
			)
			return ctx
		}).
		// ── explicit NUMA split from the driver README ──────────────────
		Assess("a claim splitting CPUs across NUMA nodes allocates on both", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// CI machines are commonly single-NUMA, so the split shape gets
			// a dedicated fake node with a fixed two-NUMA topology.
			numaNode.Annotations[draCPUAnnotation] = "8"
			numaNode.Annotations[draCPUNUMAAnnotation] = "2"
			ctx = helper.CreateNode(numaNode)(ctx, t, cfg)

			t.Log("waiting for the two-NUMA resource slice", numaSliceName)
			numaSlice := &resourcev1.ResourceSlice{}
			numaSlice.Name = numaSliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(numaSlice, func(obj k8s.Object) bool {
					return obj.(*resourcev1.ResourceSlice).Spec.Driver == draCPUDriver
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				dumpDRACPUDiagnostics(ctx, t, cfg, numaNode.Name)
				t.Fatal("two-NUMA resource slice was not published:", err)
			}
			if len(numaSlice.Spec.Devices) != 2 {
				t.Fatalf("two-NUMA slice published %d devices, want 2", len(numaSlice.Spec.Devices))
			}

			// The NUMA distribution shape from the driver README: one
			// request per NUMA node, pinned by a CEL selector.
			var requests []resourcev1.DeviceRequest
			for i := range 2 {
				request := capacityRequest(fmt.Sprintf("numa%d-cpus", i), 2)
				request.Exactly.Selectors = []resourcev1.DeviceSelector{
					{CEL: &resourcev1.CELDeviceSelector{Expression: fmt.Sprintf(`device.attributes["dra.cpu"].numaNodeID == %d`, i)}},
				}
				requests = append(requests, request)
			}
			claim := newCPUClaimFromRequests("kwok-cpu-claim-numa-split", namespace, requests...)
			pod := newCPUClaimPodWithCPUs("kwok-cpu-pod-numa-split", namespace, "fake-image", claim, map[string]string{"type": "kwok"}, 4)

			allocated := runCPUClaimLifecycle(ctx, t, client, claim, pod)
			results := allocated.Status.Allocation.Devices.Results
			if len(results) != 2 {
				t.Fatalf("expected 2 allocation results, got %d", len(results))
			}
			// All requests of a claim are satisfied by a single node; read
			// the device attributes back from that node's slice.
			allocSlice := &resourcev1.ResourceSlice{}
			if err := client.Get(ctx, results[0].Pool+"-"+draCPUDriver, "", allocSlice); err != nil {
				t.Fatal(err)
			}
			numas := map[int64]string{}
			for _, r := range results {
				if c, ok := r.ConsumedCapacity[draCPUCapacity]; !ok || c.CmpInt64(2) != 0 {
					t.Errorf("numa-split: request %q consumed %v, want 2", r.Request, r.ConsumedCapacity)
				}
				dev := deviceByName(allocSlice.Spec.Devices, r.Device)
				if dev == nil {
					t.Errorf("numa-split: allocated device %q not found in slice %q", r.Device, allocSlice.Name)
					continue
				}
				if numa := dev.Attributes["dra.cpu/numaNodeID"]; numa.IntValue == nil {
					t.Errorf("numa-split: device %q has no dra.cpu/numaNodeID attribute", r.Device)
				} else if other, dup := numas[*numa.IntValue]; dup {
					t.Errorf("numa-split: requests %q and %q both allocated on NUMA node %d", other, r.Request, *numa.IntValue)
				} else {
					numas[*numa.IntValue] = r.Request
				}
			}
			deleteClaimPod(ctx, t, client, claim, pod)
			return ctx
		}).
		// ── individual device mode allocation ──────────────────────────
		Assess("individual mode allocates distinct exclusive devices", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// A dedicated fake node with a fixed small topology publishes
			// exclusive per-CPU devices; the slice content itself is pinned
			// by the golden files in kustomize/stage/dra/cpu/testdata.
			individualNode.Annotations[draCPUAnnotation] = "8"
			individualNode.Annotations[draCPUNUMAAnnotation] = "2"
			individualNode.Annotations[draCPUModeAnnotation] = "individual"
			ctx = helper.CreateNode(individualNode)(ctx, t, cfg)

			t.Log("waiting for the individual mode resource slice", individualSliceName)
			individualSlice := &resourcev1.ResourceSlice{}
			individualSlice.Name = individualSliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(individualSlice, func(obj k8s.Object) bool {
					return obj.(*resourcev1.ResourceSlice).Spec.Driver == draCPUDriver
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				dumpDRACPUDiagnostics(ctx, t, cfg, individualNode.Name)
				t.Fatal("individual mode resource slice was not published:", err)
			}
			if len(individualSlice.Spec.Devices) != 8 {
				t.Fatalf("individual mode slice published %d devices, want 8", len(individualSlice.Spec.Devices))
			}

			// The multi-container shape from the driver's individual mode
			// example: each container mirrors and consumes its own claim,
			// restricted to individual devices via their cpuID attribute.
			claims := []*resourcev1.ResourceClaim{
				newCPUClaimFromRequests("kwok-cpu-claim-multi-container-a", namespace, countRequest(1)),
				newCPUClaimFromRequests("kwok-cpu-claim-multi-container-b", namespace, countRequest(2)),
			}
			pod := newMultiCPUClaimPod("kwok-cpu-pod-multi-container", namespace, "fake-image", map[string]string{"type": "kwok"}, claims, []int64{1, 2})

			for _, claim := range claims {
				t.Log("creating resource claim", log.KObj(claim))
				if err := client.Create(ctx, claim); err != nil {
					t.Fatal(err)
				}
			}
			t.Log("creating pod with one claim per container", log.KObj(pod))
			if err := client.Create(ctx, pod); err != nil {
				t.Fatal(err)
			}
			err = wait.For(
				conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				t.Fatal("pod did not reach Running phase:", err)
			}

			devices := map[string]string{}
			for _, claim := range claims {
				if err := client.Get(ctx, claim.Name, claim.Namespace, claim); err != nil {
					t.Fatal(err)
				}
				if claim.Status.Allocation == nil {
					t.Fatalf("resource claim %s is not allocated", claim.Name)
				}
				if len(claim.Status.ReservedFor) != 1 || claim.Status.ReservedFor[0].UID != pod.UID {
					t.Fatalf("resource claim %s is not reserved for the pod: %+v", claim.Name, claim.Status.ReservedFor)
				}
				for _, r := range claim.Status.Allocation.Devices.Results {
					if other, dup := devices[r.Device]; dup {
						t.Errorf("device %q allocated to both %s and %s", r.Device, other, claim.Name)
					}
					devices[r.Device] = claim.Name
					if r.ShareID != nil || len(r.ConsumedCapacity) != 0 {
						t.Errorf("exclusive individual device %q must have no shareID or consumed capacity", r.Device)
					}
				}
			}
			if len(devices) != 3 {
				t.Errorf("expected 3 distinct devices across both claims, got %d", len(devices))
			}

			if err := client.Delete(ctx, pod); err != nil {
				t.Fatal(err)
			}
			err = wait.For(
				conditions.New(client).ResourceDeleted(pod),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				t.Fatal("pod was not deleted:", err)
			}
			for _, claim := range claims {
				if err := client.Delete(ctx, claim); err != nil {
					t.Fatal(err)
				}
				err = wait.For(
					conditions.New(client).ResourceDeleted(claim),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("claim was not deleted:", err)
				}
			}
			return ctx
		}).
		// ── resource claim templates on both sides ─────────────────────────
		Assess("resource claim templates allocate and garbage collect on both", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			info, ok := ctx.Value(realCPUSlicesKey).(realCPUSliceInfo)
			if !ok {
				t.Fatal("requires the 'real driver publishes resource slices' assessment to have run")
			}

			// The real driver's e2e suite consumes devices exclusively
			// through ResourceClaimTemplates, so the per-pod generated
			// claims must work on both sides too.
			realTemplate := newCPUClaimTemplate("real-cpu-claim-template", namespace)
			kwokTemplate := newCPUClaimTemplate("kwok-cpu-claim-template", namespace)
			for _, template := range []*resourcev1.ResourceClaimTemplate{realTemplate, kwokTemplate} {
				t.Log("creating resource claim template", log.KObj(template))
				if err := client.Create(ctx, template); err != nil {
					t.Fatal(err)
				}
			}

			realTemplatePod := newCPUClaimTemplatePod("real-cpu-template-pod", namespace, "registry.k8s.io/pause:3.10", realTemplate.Name, map[string]string{"kubernetes.io/hostname": info.NodeName})
			kwokTemplatePod := newCPUClaimTemplatePod("kwok-cpu-template-pod", namespace, "fake-image", kwokTemplate.Name, map[string]string{"kubernetes.io/hostname": node.Name})
			pods := []*corev1.Pod{realTemplatePod, kwokTemplatePod}
			for _, pod := range pods {
				t.Log("creating pod with resource claim template", log.KObj(pod))
				if err := client.Create(ctx, pod); err != nil {
					t.Fatal(err)
				}
			}
			for _, pod := range pods {
				t.Log("waiting for pod to be running", log.KObj(pod))
				err = wait.For(
					conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("pod did not reach Running phase:", err)
				}
			}

			// The generated claims are named through the pod status.
			generatedClaims := make([]*resourcev1.ResourceClaim, 0, len(pods))
			for _, pod := range pods {
				if err := client.Get(ctx, pod.Name, pod.Namespace, pod); err != nil {
					t.Fatal(err)
				}
				var claimName string
				for _, s := range pod.Status.ResourceClaimStatuses {
					if s.Name == "cpus" && s.ResourceClaimName != nil {
						claimName = *s.ResourceClaimName
					}
				}
				if claimName == "" {
					t.Fatalf("pod %s reports no generated resource claim: %+v", pod.Name, pod.Status.ResourceClaimStatuses)
				}
				claim := &resourcev1.ResourceClaim{}
				if err := client.Get(ctx, claimName, pod.Namespace, claim); err != nil {
					t.Fatal(err)
				}
				if claim.Status.Allocation == nil {
					t.Fatalf("generated resource claim %s is not allocated", claim.Name)
				}
				if len(claim.Status.Allocation.Devices.Results) == 0 {
					t.Fatalf("generated resource claim %s has no allocation results", claim.Name)
				}
				if len(claim.Status.ReservedFor) != 1 || claim.Status.ReservedFor[0].UID != pod.UID {
					t.Fatalf("generated resource claim %s is not reserved for its pod: %+v", claim.Name, claim.Status.ReservedFor)
				}
				generatedClaims = append(generatedClaims, claim)
			}
			realResult := generatedClaims[0].Status.Allocation.Devices.Results[0]
			kwokResult := generatedClaims[1].Status.Allocation.Devices.Results[0]
			compareCPUAllocations(t, &realResult, &kwokResult)

			// Deleting the pods must garbage collect the generated claims
			// through their owner references.
			for _, pod := range pods {
				t.Log("deleting pod", log.KObj(pod))
				if err := client.Delete(ctx, pod); err != nil {
					t.Fatal(err)
				}
				err = wait.For(
					conditions.New(client).ResourceDeleted(pod),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("pod was not deleted:", err)
				}
			}
			for _, claim := range generatedClaims {
				t.Log("waiting for generated claim to be garbage collected", log.KObj(claim))
				err = wait.For(
					conditions.New(client).ResourceDeleted(claim),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("generated resource claim was not garbage collected:", err)
				}
			}
			for _, template := range []*resourcev1.ResourceClaimTemplate{realTemplate, kwokTemplate} {
				if err := client.Delete(ctx, template); err != nil {
					t.Fatal(err)
				}
				err = wait.For(
					conditions.New(client).ResourceDeleted(template),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("resource claim template was not deleted:", err)
				}
			}
			return ctx
		}).
		// ── unsatisfiable selector → pending on both sides ─────────────────
		Assess("an unsatisfiable selector leaves both pods pending", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			info, ok := ctx.Value(realCPUSlicesKey).(realCPUSliceInfo)
			if !ok {
				t.Fatal("requires the 'real driver publishes resource slices' assessment to have run")
			}

			// No device on either side lives on NUMA node 9999, so neither
			// scheduler can ever allocate the claims.
			sides := []struct {
				name     string
				image    string
				selector map[string]string
			}{
				{"unsat-cel-real", "registry.k8s.io/pause:3.10", map[string]string{"kubernetes.io/hostname": info.NodeName}},
				{"unsat-cel-simulation", "fake-image", map[string]string{"kubernetes.io/hostname": node.Name}},
			}
			claims := make([]*resourcev1.ResourceClaim, 0, len(sides))
			pods := make([]*corev1.Pod, 0, len(sides))
			for _, side := range sides {
				request := capacityRequest("cpus", 1)
				request.Exactly.Selectors = []resourcev1.DeviceSelector{
					{CEL: &resourcev1.CELDeviceSelector{Expression: `device.attributes["dra.cpu"].numaNodeID == 9999`}},
				}
				claim := newCPUClaimFromRequests(side.name, namespace, request)
				pod := newCPUClaimPod(side.name, namespace, side.image, claim, side.selector)
				t.Log("creating unsatisfiable resource claim and its pod", log.KObj(claim))
				if err := client.Create(ctx, claim); err != nil {
					t.Fatal(err)
				}
				if err := client.Create(ctx, pod); err != nil {
					t.Fatal(err)
				}
				claims = append(claims, claim)
				pods = append(pods, pod)
			}
			for _, pod := range pods {
				waitPodUnschedulable(ctx, t, client, pod)
			}
			for _, claim := range claims {
				if err := client.Get(ctx, claim.Name, claim.Namespace, claim); err != nil {
					t.Fatal(err)
				}
				if claim.Status.Allocation != nil {
					t.Errorf("resource claim %s must stay unallocated, got allocation: %+v", claim.Name, claim.Status.Allocation)
				}
			}
			for i := range claims {
				deleteClaimPod(ctx, t, client, claims[i], pods[i])
			}
			return ctx
		}).
		// ── capacity exhaustion blocks scheduling until release ────────────
		Assess("exhausted capacity leaves a pod pending until released", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// The dedicated fake node publishes a single 4-CPU device,
			// bounding the allocatable capacity deterministically.
			ctx = helper.CreateNode(capNode)(ctx, t, cfg)

			t.Log("waiting for the capacity-bounded resource slice", capSliceName)
			capSlice := &resourcev1.ResourceSlice{}
			capSlice.Name = capSliceName
			err = wait.For(
				conditions.New(client).ResourceMatch(capSlice, func(obj k8s.Object) bool {
					return obj.(*resourcev1.ResourceSlice).Spec.Driver == draCPUDriver
				}),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				dumpDRACPUDiagnostics(ctx, t, cfg, capNode.Name)
				t.Fatal("capacity-bounded resource slice was not published:", err)
			}
			if len(capSlice.Spec.Devices) != 1 {
				t.Fatalf("capacity-bounded slice published %d devices, want 1", len(capSlice.Spec.Devices))
			}
			if capacity, ok := capSlice.Spec.Devices[0].Capacity[draCPUCapacity]; !ok || capacity.Value.CmpInt64(4) != 0 {
				t.Fatalf("capacity-bounded device published capacity %v, want %s=4", capSlice.Spec.Devices[0].Capacity, draCPUCapacity)
			}

			capSelector := map[string]string{"kubernetes.io/hostname": capNode.Name}
			fillClaim := newCPUClaimFromRequests("kwok-cpu-claim-cap-fill", namespace, capacityRequest("cpus", 3))
			fillPod := newCPUClaimPodWithCPUs("kwok-cpu-pod-cap-fill", namespace, "fake-image", fillClaim, capSelector, 3)
			runCPUClaimLifecycle(ctx, t, client, fillClaim, fillPod)

			// 2 more CPUs do not fit into the remaining 1.
			pendingClaim := newCPUClaimFromRequests("kwok-cpu-claim-cap-wait", namespace, capacityRequest("cpus", 2))
			pendingPod := newCPUClaimPodWithCPUs("kwok-cpu-pod-cap-wait", namespace, "fake-image", pendingClaim, capSelector, 2)
			t.Log("creating resource claim", log.KObj(pendingClaim))
			if err := client.Create(ctx, pendingClaim); err != nil {
				t.Fatal(err)
			}
			t.Log("creating pod contending for the exhausted capacity", log.KObj(pendingPod))
			if err := client.Create(ctx, pendingPod); err != nil {
				t.Fatal(err)
			}
			waitPodUnschedulable(ctx, t, client, pendingPod)
			if err := client.Get(ctx, pendingClaim.Name, pendingClaim.Namespace, pendingClaim); err != nil {
				t.Fatal(err)
			}
			if pendingClaim.Status.Allocation != nil {
				t.Errorf("resource claim %s must stay unallocated while the capacity is exhausted, got allocation: %+v", pendingClaim.Name, pendingClaim.Status.Allocation)
			}

			// Releasing the blocking claim must return its capacity and let
			// the pending pod through.
			deleteClaimPod(ctx, t, client, fillClaim, fillPod)
			t.Log("waiting for the pending pod to run after the capacity release", log.KObj(pendingPod))
			err = wait.For(
				conditions.New(client).PodPhaseMatch(pendingPod, corev1.PodRunning),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				t.Fatal("pending pod did not run after the capacity was released:", err)
			}
			if err := client.Get(ctx, pendingClaim.Name, pendingClaim.Namespace, pendingClaim); err != nil {
				t.Fatal(err)
			}
			if pendingClaim.Status.Allocation == nil {
				t.Fatal("resource claim was not allocated after the capacity was released")
			}
			deleteClaimPod(ctx, t, client, pendingClaim, pendingPod)
			return ctx
		}).
		// ── one claim shared by two pods (simulation only) ──────────────────
		// The real driver rejects a shared claim's second pod at the node
		// level (its e2e sharing test); kwok has no node agent, so both pods
		// run and only the scheduler-level ReservedFor semantics are checked.
		Assess("two pods share one claim on the simulation", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			sharedClaim := newCPUClaim("kwok-cpu-claim-shared", namespace)
			capSelector := map[string]string{"kubernetes.io/hostname": capNode.Name}
			pods := []*corev1.Pod{
				newCPUClaimPod("kwok-cpu-pod-shared-a", namespace, "fake-image", sharedClaim, capSelector),
				newCPUClaimPod("kwok-cpu-pod-shared-b", namespace, "fake-image", sharedClaim, capSelector),
			}

			t.Log("creating shared resource claim", log.KObj(sharedClaim))
			if err := client.Create(ctx, sharedClaim); err != nil {
				t.Fatal(err)
			}
			for _, pod := range pods {
				t.Log("creating pod sharing the claim", log.KObj(pod))
				if err := client.Create(ctx, pod); err != nil {
					t.Fatal(err)
				}
			}
			for _, pod := range pods {
				t.Log("waiting for pod to be running", log.KObj(pod))
				err = wait.For(
					conditions.New(client).PodPhaseMatch(pod, corev1.PodRunning),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("pod did not reach Running phase:", err)
				}
			}

			if err := client.Get(ctx, sharedClaim.Name, sharedClaim.Namespace, sharedClaim); err != nil {
				t.Fatal(err)
			}
			if sharedClaim.Status.Allocation == nil {
				t.Fatal("shared resource claim is not allocated")
			}
			if got := len(sharedClaim.Status.Allocation.Devices.Results); got != 1 {
				t.Errorf("shared claim must have exactly 1 device result, got %d", got)
			}
			if len(sharedClaim.Status.ReservedFor) != 2 {
				t.Fatalf("shared claim must be reserved for exactly 2 pods: %+v", sharedClaim.Status.ReservedFor)
			}
			reserved := map[types.UID]bool{}
			for _, ref := range sharedClaim.Status.ReservedFor {
				reserved[ref.UID] = true
			}
			for _, pod := range pods {
				if !reserved[pod.UID] {
					t.Errorf("shared claim is not reserved for pod %s (uid %s): %+v", pod.Name, pod.UID, sharedClaim.Status.ReservedFor)
				}
			}

			for _, pod := range pods {
				if err := client.Delete(ctx, pod); err != nil {
					t.Fatal(err)
				}
				err = wait.For(
					conditions.New(client).ResourceDeleted(pod),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("pod was not deleted:", err)
				}
			}
			if err := client.Delete(ctx, sharedClaim); err != nil {
				t.Fatal(err)
			}
			err = wait.For(
				conditions.New(client).ResourceDeleted(sharedClaim),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				t.Fatal("claim was not deleted:", err)
			}
			return ctx
		}).
		// ── individual device exhaustion blocks scheduling until release ───
		Assess("exhausted individual devices leave a pod pending until released", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			// The individual mode assessment cleaned up after itself, so all
			// 8 exclusive devices of its node are free again.
			indSelector := map[string]string{"kubernetes.io/hostname": individualNode.Name}
			allClaim := newCPUClaimFromRequests("kwok-cpu-claim-ind-all", namespace, countRequest(8))
			allPod := newCPUClaimPodWithCPUs("kwok-cpu-pod-ind-all", namespace, "fake-image", allClaim, indSelector, 8)
			allAllocated := runCPUClaimLifecycle(ctx, t, client, allClaim, allPod)
			if got := len(allAllocated.Status.Allocation.Devices.Results); got != 8 {
				t.Fatalf("expected the claim to allocate all 8 individual devices, got %d results", got)
			}

			oneClaim := newCPUClaimFromRequests("kwok-cpu-claim-ind-one", namespace, countRequest(1))
			onePod := newCPUClaimPod("kwok-cpu-pod-ind-one", namespace, "fake-image", oneClaim, indSelector)
			t.Log("creating resource claim", log.KObj(oneClaim))
			if err := client.Create(ctx, oneClaim); err != nil {
				t.Fatal(err)
			}
			t.Log("creating pod contending for the exhausted devices", log.KObj(onePod))
			if err := client.Create(ctx, onePod); err != nil {
				t.Fatal(err)
			}
			waitPodUnschedulable(ctx, t, client, onePod)
			if err := client.Get(ctx, oneClaim.Name, oneClaim.Namespace, oneClaim); err != nil {
				t.Fatal(err)
			}
			if oneClaim.Status.Allocation != nil {
				t.Errorf("resource claim %s must stay unallocated while all devices are taken, got allocation: %+v", oneClaim.Name, oneClaim.Status.Allocation)
			}

			// Releasing the blocking claim must free its devices and let the
			// pending pod through.
			deleteClaimPod(ctx, t, client, allClaim, allPod)
			t.Log("waiting for the pending pod to run after the devices are released", log.KObj(onePod))
			err = wait.For(
				conditions.New(client).PodPhaseMatch(onePod, corev1.PodRunning),
				wait.WithContext(ctx),
				wait.WithTimeout(waitTimeout),
			)
			if err != nil {
				t.Fatal("pending pod did not run after the devices were released:", err)
			}
			if err := client.Get(ctx, oneClaim.Name, oneClaim.Namespace, oneClaim); err != nil {
				t.Fatal(err)
			}
			if oneClaim.Status.Allocation == nil {
				t.Fatal("resource claim was not allocated after the devices were released")
			}
			deleteClaimPod(ctx, t, client, oneClaim, onePod)
			return ctx
		}).
		// ── fake node deletion → simulated slices garbage collected ────────
		Assess("node deletion garbage collects the simulated resource slices", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}

			for _, n := range []*corev1.Node{node, individualNode, numaNode, capNode} {
				t.Log("deleting node", n.Name)
				if err = client.Delete(ctx, n); err != nil {
					t.Fatal(err)
				}
				err = wait.For(
					conditions.New(client).ResourceDeleted(n),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("node was not deleted:", err)
				}
			}

			for _, name := range []string{simSliceName, individualSliceName, numaSliceName, capSliceName} {
				simSlice := &resourcev1.ResourceSlice{}
				simSlice.Name = name
				err = wait.For(
					conditions.New(client).ResourceDeleted(simSlice),
					wait.WithContext(ctx),
					wait.WithTimeout(waitTimeout),
				)
				if err != nil {
					t.Fatal("simulated resource slice was not garbage collected:", err)
				}
				t.Log("simulated resource slice garbage collected", name)
			}
			return ctx
		}).
		// ── teardown ───────────────────────────────────────────────────────
		Teardown(func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			client, err := resources.New(cfg.Client().RESTConfig())
			if err != nil {
				t.Fatal(err)
			}
			// Best-effort cleanup in case an assessment failed midway,
			// including the deterministically named objects of the later
			// assessments.
			pods := []*corev1.Pod{realPod, kwokPod, realSharePod, kwokSharePod}
			for _, name := range []string{
				"cel-selector-real", "cel-selector-simulation",
				"match-attribute-real", "match-attribute-simulation",
				"kwok-cpu-pod-numa-split", "kwok-cpu-pod-multi-container",
				"real-cpu-template-pod", "kwok-cpu-template-pod",
				"unsat-cel-real", "unsat-cel-simulation",
				"kwok-cpu-pod-cap-fill", "kwok-cpu-pod-cap-wait",
				"kwok-cpu-pod-shared-a", "kwok-cpu-pod-shared-b",
				"kwok-cpu-pod-ind-all", "kwok-cpu-pod-ind-one",
			} {
				pods = append(pods, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
			}
			for _, pod := range pods {
				if err = client.Delete(ctx, pod); err != nil && !apierrors.IsNotFound(err) {
					t.Log("warning: failed to delete pod", log.KObj(pod), err)
				}
			}
			claims := []*resourcev1.ResourceClaim{realClaim, kwokClaim, realShareClaim, kwokShareClaim}
			for _, name := range []string{
				"cel-selector-real", "cel-selector-simulation",
				"match-attribute-real", "match-attribute-simulation",
				"kwok-cpu-claim-numa-split",
				"kwok-cpu-claim-multi-container-a", "kwok-cpu-claim-multi-container-b",
				"unsat-cel-real", "unsat-cel-simulation",
				"kwok-cpu-claim-cap-fill", "kwok-cpu-claim-cap-wait",
				"kwok-cpu-claim-shared",
				"kwok-cpu-claim-ind-all", "kwok-cpu-claim-ind-one",
			} {
				claims = append(claims, &resourcev1.ResourceClaim{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}})
			}
			for _, claim := range claims {
				if err = client.Delete(ctx, claim); err != nil && !apierrors.IsNotFound(err) {
					t.Log("warning: failed to delete resource claim", log.KObj(claim), err)
				}
			}
			// Claims generated from the templates are garbage collected with
			// their pods; the templates themselves must be removed.
			for _, name := range []string{"real-cpu-claim-template", "kwok-cpu-claim-template"} {
				template := &resourcev1.ResourceClaimTemplate{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
				if err = client.Delete(ctx, template); err != nil && !apierrors.IsNotFound(err) {
					t.Log("warning: failed to delete resource claim template", log.KObj(template), err)
				}
			}
			for _, n := range []*corev1.Node{node, individualNode, numaNode, capNode} {
				if err = client.Delete(ctx, n); err != nil && !apierrors.IsNotFound(err) {
					t.Log("warning: failed to delete node", n.Name, err)
				}
			}
			return ctx
		})
}

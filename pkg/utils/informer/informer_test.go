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

package informer

import (
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestInformerSync(t *testing.T) {
	now := time.Unix(0, 0)

	fakeClient := fake.NewClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease0",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease0"),
				RenewTime:      new(metav1.NewMicroTime(now)),
			},
		},
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease1",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease1"),
				RenewTime:      new(metav1.NewMicroTime(now)),
			},
		},
	)
	cli := fakeClient.CoordinationV1().Leases("default")
	informer := NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](cli)

	ctx := t.Context()

	events := make(chan Event[*coordinationv1.Lease], 100)
	_ = informer.Sync(ctx, Option{}, events)

	if size := len(events); size != 2 {
		t.Error("expected 2 events, got", size)
	}

	event, ok := <-events
	if !ok {
		t.Error("expected event, got", ok)
	}
	if event.Object.Name != "lease0" {
		t.Error("expected lease0, got", event.Object.Name)
	}
	if event.Type != Sync {
		t.Error("expected Sync event, got", event.Type)
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	}
	if event.Object.Name != "lease1" {
		t.Error("expected lease1, got", event.Object.Name)
	}
	if event.Type != Sync {
		t.Error("expected Sync event, got", event.Type)
	}
}

func TestInformerWatch(t *testing.T) {
	now := time.Unix(1, 0)

	fakeClient := fake.NewClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease0",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease0"),
				RenewTime:      new(metav1.NewMicroTime(now)),
			},
		},
	)
	cli := fakeClient.CoordinationV1().Leases("default")
	informer := NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](cli)

	ctx := t.Context()

	events := make(chan Event[*coordinationv1.Lease], 100)
	_ = informer.Watch(ctx, Option{}, events)

	time.Sleep(1 * time.Second)

	_, _ = cli.Create(ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "lease1",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease1"),
				RenewTime:      new(metav1.NewMicroTime(now)),
			},
		},
		metav1.CreateOptions{},
	)

	time.Sleep(1 * time.Second)

	_, _ = cli.Update(ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "lease1",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease1"),
				RenewTime:      new(metav1.NewMicroTime(now.Add(1 * time.Second))),
			},
		},
		metav1.UpdateOptions{},
	)

	time.Sleep(1 * time.Second)

	_ = cli.Delete(ctx,
		"lease1",
		metav1.DeleteOptions{},
	)

	time.Sleep(1 * time.Second)

	var got []Event[*coordinationv1.Lease]
	for len(events) != 0 {
		got = append(got, <-events)
	}

	hasLease0Initial := false
	hasLease1Added := false
	hasLease1Modified := false
	hasLease1Deleted := false
	for _, event := range got {
		if event.Object.Name == "lease0" && (event.Type == Sync || event.Type == Added) {
			hasLease0Initial = true
		}
		if event.Object.Name == "lease1" && event.Type == Added && event.Object.Spec.RenewTime.Time.Equal(now) {
			hasLease1Added = true
		}
		if event.Object.Name == "lease1" && event.Type == Modified && event.Object.Spec.RenewTime.Time.Equal(now.Add(1*time.Second)) {
			hasLease1Modified = true
		}
		if event.Object.Name == "lease1" && event.Type == Deleted {
			hasLease1Deleted = true
		}
	}

	if !hasLease0Initial {
		t.Error("expected at least one initial event for lease0")
	}
	if !hasLease1Added {
		t.Error("expected add event for lease1")
	}
	if !hasLease1Modified {
		t.Error("expected modify event for lease1")
	}
	if !hasLease1Deleted {
		t.Error("expected delete event for lease1")
	}
}

func TestInformerWatchWithCache(t *testing.T) {
	now := time.Unix(1, 0)

	fakeClient := fake.NewClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease0",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease0"),
				RenewTime:      new(metav1.NewMicroTime(now)),
			},
		},
	)
	cli := fakeClient.CoordinationV1().Leases("default")
	informer := NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](cli)

	ctx := t.Context()

	events := make(chan Event[*coordinationv1.Lease], 100)
	getter, _ := informer.WatchWithCache(ctx, Option{}, events)

	time.Sleep(1 * time.Second)

	_, _ = cli.Create(ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "lease1",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease1"),
				RenewTime:      new(metav1.NewMicroTime(now)),
			},
		},
		metav1.CreateOptions{},
	)

	time.Sleep(1 * time.Second)
	lease, ok := waitForLeaseWithRenewTime(getter, now)
	if !ok {
		t.Error("expected lease1 in cache")
	} else if !lease.Spec.RenewTime.Time.Equal(now) {
		t.Error("expected lease1 with renew time", now)
	}

	_, _ = cli.Update(ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "lease1",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease1"),
				RenewTime:      new(metav1.NewMicroTime(now.Add(1 * time.Second))),
			},
		},
		metav1.UpdateOptions{},
	)

	time.Sleep(1 * time.Second)
	lease, ok = waitForLeaseWithRenewTime(getter, now.Add(1*time.Second))
	if !ok {
		t.Error("expected lease1 in cache")
	} else if !lease.Spec.RenewTime.Time.Equal(now.Add(1 * time.Second)) {
		t.Error("expected lease1 with renew time", now.Add(1*time.Second))
	}

	_, _ = cli.Update(ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "lease1",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: new("lease1"),
				RenewTime:      new(metav1.NewMicroTime(now.Add(2 * time.Second))),
			},
		},
		metav1.UpdateOptions{},
	)

	time.Sleep(1 * time.Second)
	lease, ok = waitForLeaseWithRenewTime(getter, now.Add(2*time.Second))
	if !ok {
		t.Error("expected lease1 in cache")
	} else if !lease.Spec.RenewTime.Time.Equal(now.Add(2 * time.Second)) {
		t.Error("expected lease1 with renew time", now.Add(2*time.Second))
	}

	_ = cli.Delete(ctx,
		"lease1",
		metav1.DeleteOptions{},
	)

	time.Sleep(1 * time.Second)
	if _, ok := getter.GetWithNamespace("lease1", "default"); ok {
		t.Error("expected lease1 removed from cache")
	}

	var got []Event[*coordinationv1.Lease]
	for len(events) != 0 {
		got = append(got, <-events)
	}

	hasLease0Initial := false
	hasLease1Added := false
	hasLease1Modified1 := false
	hasLease1Modified2 := false
	hasLease1Deleted := false
	for _, event := range got {
		if event.Object.Name == "lease0" && (event.Type == Sync || event.Type == Added) {
			hasLease0Initial = true
		}
		if event.Object.Name == "lease1" && event.Type == Added && event.Object.Spec.RenewTime.Time.Equal(now) {
			hasLease1Added = true
		}
		if event.Object.Name == "lease1" && event.Type == Modified && event.Object.Spec.RenewTime.Time.Equal(now.Add(1*time.Second)) {
			hasLease1Modified1 = true
		}
		if event.Object.Name == "lease1" && event.Type == Modified && event.Object.Spec.RenewTime.Time.Equal(now.Add(2*time.Second)) {
			hasLease1Modified2 = true
		}
		if event.Object.Name == "lease1" && event.Type == Deleted {
			hasLease1Deleted = true
		}
	}

	if !hasLease0Initial {
		t.Error("expected at least one initial event for lease0")
	}
	if !hasLease1Added {
		t.Error("expected add event for lease1")
	}
	if !hasLease1Modified1 {
		t.Error("expected first modify event for lease1")
	}
	if !hasLease1Modified2 {
		t.Error("expected second modify event for lease1")
	}
	if !hasLease1Deleted {
		t.Error("expected delete event for lease1")
	}
}

func waitForLeaseWithRenewTime(getter Getter[*coordinationv1.Lease], renewTime time.Time) (*coordinationv1.Lease, bool) {
	timeout := time.After(3 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		lease, ok := getter.GetWithNamespace("lease1", "default")
		if ok && lease.Spec.RenewTime.Time.Equal(renewTime) {
			return lease, true
		}
		select {
		case <-timeout:
			return nil, false
		case <-ticker.C:
		}
	}
}

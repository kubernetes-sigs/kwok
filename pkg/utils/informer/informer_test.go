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
	"context"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/kwok/pkg/utils/format"
)

func TestInformerSync(t *testing.T) {
	now := time.Unix(0, 0)

	fakeClient := fake.NewSimpleClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease0",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: format.Ptr("lease0"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now)),
			},
		},
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease1",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: format.Ptr("lease1"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now)),
			},
		},
	)
	cli := fakeClient.CoordinationV1().Leases("default")
	informer := NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](cli)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	fakeClient := fake.NewSimpleClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease0",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: format.Ptr("lease0"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now)),
			},
		},
	)
	cli := fakeClient.CoordinationV1().Leases("default")
	informer := NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](cli)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan Event[*coordinationv1.Lease], 100)
	_ = informer.Watch(ctx, Option{}, events)

	time.Sleep(1 * time.Second)

	_, _ = cli.Create(ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "lease1",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: format.Ptr("lease1"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now)),
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
				HolderIdentity: format.Ptr("lease1"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now.Add(1 * time.Second))),
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

	if size := len(events); size != 4 {
		t.Error("expected 4 events, got", size)
	}

	event, ok := <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease0" {
			t.Error("expected lease0, got", event.Object.Name)
		}
		if event.Type != Sync {
			t.Error("expected Sync event, got", event.Type)
		}
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease1" {
			t.Error("expected lease1, got", event.Object.Name)
		}
		if event.Type != Added {
			t.Error("expected Add event, got", event.Type)
		}
		if event.Object.Spec.RenewTime.Time != now {
			t.Error("expected Add event, got", event.Type)
		}
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease1" {
			t.Error("expected lease1, got", event.Object.Name)
		}
		if event.Type != Modified || event.Object.Name != "lease1" {
			t.Error("expected Modify event, got", event.Type)
		}
		if event.Object.Spec.RenewTime.Time != now.Add(1*time.Second) {
			t.Error("expected Modify event, got", event.Type)
		}
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease1" {
			t.Error("expected lease1, got", event.Object.Name)
		}
		if event.Type != Deleted {
			t.Error("expected Delete event, got", event.Type)
		}
	}
}

func TestInformerWatchWithCache(t *testing.T) {
	now := time.Unix(1, 0)

	fakeClient := fake.NewSimpleClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease0",
				Namespace: "default",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: format.Ptr("lease0"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now)),
			},
		},
	)
	cli := fakeClient.CoordinationV1().Leases("default")
	informer := NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](cli)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events := make(chan Event[*coordinationv1.Lease], 100)
	getter, _ := informer.WatchWithCache(ctx, Option{}, events)

	time.Sleep(1 * time.Second)

	_, _ = cli.Create(ctx,
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name: "lease1",
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: format.Ptr("lease1"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now)),
			},
		},
		metav1.CreateOptions{},
	)

	time.Sleep(1 * time.Second)
	lease, ok := getter.GetWithNamespace("lease1", "default")
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
				HolderIdentity: format.Ptr("lease1"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now.Add(1 * time.Second))),
			},
		},
		metav1.UpdateOptions{},
	)

	time.Sleep(1 * time.Second)
	lease, ok = getter.GetWithNamespace("lease1", "default")
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
				HolderIdentity: format.Ptr("lease1"),
				RenewTime:      format.Ptr(metav1.NewMicroTime(now.Add(2 * time.Second))),
			},
		},
		metav1.UpdateOptions{},
	)

	time.Sleep(1 * time.Second)
	lease, ok = getter.GetWithNamespace("lease1", "default")
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

	if size := len(events); size != 5 {
		t.Error("expected 5 events, got", size)
	}

	event, ok := <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease0" {
			t.Error("expected lease0, got", event.Object.Name)
		}
		if event.Type != Added {
			t.Error("expected Sync event, got", event.Type)
		}
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease1" {
			t.Error("expected lease1, got", event.Object.Name)
		}
		if event.Type != Added {
			t.Error("expected Add event, got", event.Type)
		}
		if event.Object.Spec.RenewTime.Time != now {
			t.Error("expected Add event, got", event.Type)
		}
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease1" {
			t.Error("expected lease1, got", event.Object.Name)
		}
		if event.Type != Modified || event.Object.Name != "lease1" {
			t.Error("expected Modify event, got", event.Type)
		}
		if event.Object.Spec.RenewTime.Time != now.Add(1*time.Second) {
			t.Error("expected Modify event, got", event.Type)
		}
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease1" {
			t.Error("expected lease1, got", event.Object.Name)
		}
		if event.Type != Modified || event.Object.Name != "lease1" {
			t.Error("expected Modify event, got", event.Type)
		}
		if event.Object.Spec.RenewTime.Time != now.Add(2*time.Second) {
			t.Error("expected Modify event, got", event.Type)
		}
	}

	event, ok = <-events
	if !ok {
		t.Error("expected event, got", ok)
	} else {
		if event.Object.Name != "lease1" {
			t.Error("expected lease1, got", event.Object.Name)
		}
		if event.Type != Deleted {
			t.Error("expected Delete event, got", event.Type)
		}
	}
}

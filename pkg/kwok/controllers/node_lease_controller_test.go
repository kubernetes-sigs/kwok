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

package controllers

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/informer"
)

func TestNodeLeaseController(t *testing.T) {
	now := time.Now()
	clientset := fake.NewSimpleClientset(
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease1",
				Namespace: corev1.NamespaceNodeLease,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       format.Ptr("lease1"),
				RenewTime:            format.Ptr(metav1.NewMicroTime(now.Add(-61 * time.Second))),
				LeaseDurationSeconds: format.Ptr(int32(60)),
			},
		},
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease2",
				Namespace: corev1.NamespaceNodeLease,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       format.Ptr("lease2"),
				RenewTime:            format.Ptr(metav1.NewMicroTime(now)),
				LeaseDurationSeconds: format.Ptr(int32(60)),
			},
		},
		&coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "lease3",
				Namespace: corev1.NamespaceNodeLease,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity:       format.Ptr("lease3"),
				RenewTime:            format.Ptr(metav1.NewMicroTime(now.Add(-61 * time.Second))),
				LeaseDurationSeconds: format.Ptr(int32(60)),
			},
		},
	)

	nodeLeases, err := NewNodeLeaseController(NodeLeaseControllerConfig{
		TypedClient:          clientset,
		HolderIdentity:       "test",
		LeaseDurationSeconds: 40,
		LeaseParallelism:     2,
		RenewInterval:        10 * time.Second,
		RenewIntervalJitter:  0.04,
	})
	if err != nil {
		t.Fatal(fmt.Errorf("new node leases controller error: %w", err))
	}

	ctx := context.Background()
	ctx = log.NewContext(ctx, log.NewLogger(os.Stderr, log.LevelDebug))
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	t.Cleanup(func() {
		cancel()
		time.Sleep(time.Second)
	})

	nodeLeasesCh := make(chan informer.Event[*coordinationv1.Lease], 1)
	nodeLeasesCli := clientset.CoordinationV1().Leases(corev1.NamespaceNodeLease)
	nodesInformer := informer.NewInformer[*coordinationv1.Lease, *coordinationv1.LeaseList](nodeLeasesCli)
	err = nodesInformer.Watch(ctx, informer.Option{}, nodeLeasesCh)
	if err != nil {
		t.Fatal(fmt.Errorf("watch node leases error: %w", err))
	}

	time.Sleep(1 * time.Second)

	err = nodeLeases.Start(ctx, nodeLeasesCh)
	if err != nil {
		t.Fatal(fmt.Errorf("start node leases controller error: %w", err))
	}

	nodeLeases.TryHold("lease0")
	nodeLeases.TryHold("lease1")
	nodeLeases.TryHold("lease2")

	time.Sleep(1 * time.Second)

	if !nodeLeases.Held("lease0") {
		t.Error("lease0 not held")
	}

	if !nodeLeases.Held("lease1") {
		t.Error("lease1 not held")
	}

	if nodeLeases.Held("lease2") {
		t.Error("lease2 held")
	}

	if nodeLeases.Held("lease3") {
		t.Error("lease3 held")
	}

	if nodeLeases.Held("lease4") {
		t.Error("lease4 held")
	}

	_ = clientset.CoordinationV1().Leases(corev1.NamespaceNodeLease).Delete(ctx, "lease1", metav1.DeleteOptions{})
	time.Sleep(2 * time.Second)

	if !nodeLeases.Held("lease0") {
		t.Error("lease0 not held")
	}

	if nodeLeases.Held("lease1") {
		t.Error("lease1 held")
	}

	if nodeLeases.Held("lease3") {
		t.Error("lease3 held")
	}

	if nodeLeases.Held("lease4") {
		t.Error("lease4 held")
	}
}

func Test_tryAcquireOrRenew(t *testing.T) {
	now := time.Now()
	type args struct {
		lease          *coordinationv1.Lease
		holderIdentity string
		now            time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "holder self",
			args: args{
				lease: &coordinationv1.Lease{
					Spec: coordinationv1.LeaseSpec{
						HolderIdentity: format.Ptr("test"),
					},
				},
				holderIdentity: "test",
				now:            now,
			},
			want: true,
		},
		{
			name: "holder not self",
			args: args{
				lease: &coordinationv1.Lease{
					Spec: coordinationv1.LeaseSpec{
						HolderIdentity:       format.Ptr("test"),
						LeaseDurationSeconds: format.Ptr(int32(10)),
						RenewTime:            format.Ptr(metav1.NewMicroTime(now.Add(-5 * time.Second))),
					},
				},
				holderIdentity: "test-new",
				now:            now,
			},
			want: false,
		},
		{
			name: "holder not self but expired",
			args: args{
				lease: &coordinationv1.Lease{
					Spec: coordinationv1.LeaseSpec{
						HolderIdentity:       format.Ptr("test"),
						LeaseDurationSeconds: format.Ptr(int32(10)),
						RenewTime:            format.Ptr(metav1.NewMicroTime(now.Add(-11 * time.Second))),
					},
				},
				holderIdentity: "test-new",
				now:            now,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tryAcquireOrRenew(tt.args.lease, tt.args.holderIdentity, tt.args.now); got != tt.want {
				t.Errorf("tryAcquireOrRenew() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNextTryDuration(t *testing.T) {
	tests := []struct {
		name           string
		renewInterval  time.Duration
		expire         time.Duration
		hold           bool
		expectedResult time.Duration
	}{
		{
			name:           "Lease expired",
			renewInterval:  10 * time.Second,
			expire:         -1 * time.Second,
			hold:           true,
			expectedResult: 0,
		},
		{
			name:           "Lease not held",
			renewInterval:  10 * time.Second,
			expire:         5 * time.Second,
			hold:           false,
			expectedResult: 5 * time.Second,
		},
		{
			name:           "Lease held, renew interval greater than expire time",
			renewInterval:  10 * time.Second,
			expire:         5 * time.Second,
			hold:           true,
			expectedResult: 4 * time.Second,
		},
		{
			name:           "Lease held, renew interval less than expire time",
			renewInterval:  5 * time.Second,
			expire:         10 * time.Second,
			hold:           true,
			expectedResult: 5 * time.Second,
		},
		{
			name:           "Lease held, renew interval equals expire time",
			renewInterval:  5 * time.Second,
			expire:         5 * time.Second,
			hold:           true,
			expectedResult: 4 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nextTryDuration(tt.renewInterval, tt.expire, tt.hold)
			if result != tt.expectedResult {
				t.Errorf("nextTryDuration() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestExpireTime(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		lease    *coordinationv1.Lease
		wantTime time.Time
		wantBool bool
	}{
		{
			name:     "Lease with missing fields",
			lease:    &coordinationv1.Lease{},
			wantTime: time.Time{},
			wantBool: false,
		},
		{
			name: "Lease with all fields",
			lease: &coordinationv1.Lease{
				Spec: coordinationv1.LeaseSpec{
					HolderIdentity:       format.Ptr("test"),
					LeaseDurationSeconds: format.Ptr(int32(10)),
					RenewTime:            format.Ptr(metav1.NewMicroTime(now)),
				},
			},
			wantTime: now.Add(10 * time.Second),
			wantBool: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotBool := expireTime(tt.lease)
			if !reflect.DeepEqual(gotTime, tt.wantTime) {
				t.Errorf("expireTime() gotTime = %v, want %v", gotTime, tt.wantTime)
			}
			if gotBool != tt.wantBool {
				t.Errorf("expireTime() gotBool = %v, want %v", gotBool, tt.wantBool)
			}
		})
	}
}

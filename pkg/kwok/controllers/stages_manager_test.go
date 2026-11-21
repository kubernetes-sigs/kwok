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

package controllers

import (
	"context"
	"errors"
	"testing"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/config/resources"
	"sigs.k8s.io/kwok/pkg/utils/lifecycle"
)

func TestStagesManagerCancelsFailedStarts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ref := internalversion.StageResourceRef{Kind: "Pod"}
	getter := &stagesManagerTestGetter{
		Getter: resources.NewStaticGetter([]*internalversion.Stage{{
			Spec: internalversion.StageSpec{ResourceRef: ref},
		}}),
	}
	var attempts []context.Context
	manager := NewStagesManager(StagesManagerConfig{
		StageGetter: getter,
		StartFunc: func(ctx context.Context, _ internalversion.StageResourceRef, _ resources.Getter[lifecycle.Lifecycle]) error {
			attempts = append(attempts, ctx)
			return errors.New("controller startup failed")
		},
	})

	for attempt := 1; attempt <= 2; attempt++ {
		manager.manage(ctx)
		if len(attempts) != attempt {
			t.Fatalf("startup attempts = %d, want %d", len(attempts), attempt)
		}
		if err := attempts[attempt-1].Err(); !errors.Is(err, context.Canceled) {
			t.Errorf("failed startup attempt %d context error = %v, want context.Canceled", attempt, err)
		}
		if err := ctx.Err(); err != nil {
			t.Fatalf("parent context was canceled: %v", err)
		}
	}
}

func TestStagesManagerKeepsSuccessfulRetryRunningUntilRemoval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ref := internalversion.StageResourceRef{Kind: "Pod"}
	getter := &stagesManagerTestGetter{
		Getter: resources.NewStaticGetter([]*internalversion.Stage{{
			Spec: internalversion.StageSpec{ResourceRef: ref},
		}}),
	}
	var attempts []context.Context
	manager := NewStagesManager(StagesManagerConfig{
		StageGetter: getter,
		StartFunc: func(ctx context.Context, _ internalversion.StageResourceRef, _ resources.Getter[lifecycle.Lifecycle]) error {
			attempts = append(attempts, ctx)
			if len(attempts) == 1 {
				return errors.New("controller startup failed")
			}
			return nil
		},
	})

	manager.manage(ctx)
	manager.manage(ctx)
	if len(attempts) != 2 {
		t.Fatalf("startup attempts = %d, want 2", len(attempts))
	}
	if err := attempts[0].Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("failed startup context error = %v, want context.Canceled", err)
	}
	if err := attempts[1].Err(); err != nil {
		t.Fatalf("successful retry context was canceled: %v", err)
	}

	manager.manage(ctx)
	if len(attempts) != 2 {
		t.Fatalf("running controller was restarted: startup attempts = %d, want 2", len(attempts))
	}
	if err := attempts[1].Err(); err != nil {
		t.Fatalf("running controller context was canceled: %v", err)
	}

	getter.Getter = resources.NewStaticGetter([]*internalversion.Stage{})
	manager.manage(ctx)
	if err := attempts[1].Err(); !errors.Is(err, context.Canceled) {
		t.Errorf("removed controller context error = %v, want context.Canceled", err)
	}
	if err := ctx.Err(); err != nil {
		t.Errorf("parent context was canceled: %v", err)
	}
}

type stagesManagerTestGetter struct {
	resources.Getter[[]*internalversion.Stage]
}

func (*stagesManagerTestGetter) Start(context.Context) error { return nil }

func (*stagesManagerTestGetter) Sync() <-chan struct{} { return nil }

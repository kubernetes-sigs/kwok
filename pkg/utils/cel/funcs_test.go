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

package cel

import (
	"testing"
	"time"

	"github.com/google/cel-go/common/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTimeNow(t *testing.T) {
	refVal := runAndCheckExpression(t, "Now()", types.TimestampType)

	actual := refVal.Value().(time.Time)

	if !expectedTime.Equal(actual) {
		t.Errorf("expected %v, got %v", expectedTime, actual)
	}
}

func TestMathRand(t *testing.T) {
	refVal := runAndCheckExpression(t, "Rand()", types.DoubleType)
	actual := refVal.Value().(float64)

	if actual < 0 || actual > 1 {
		t.Errorf("expected value between 0 and 1, got %v", refVal.Value())
	}
}

func TestUnixSecond(t *testing.T) {
	refVal := runAndCheckExpressionWithData(t, "UnixSecond(time)", map[string]any{
		"time": expectedTime,
	}, types.DoubleType)

	actual := refVal.Value().(float64)
	expected := float64(expectedTime.Unix())

	compareFloat64(t, expected, actual)
}

func TestSinceSecond(t *testing.T) {
	n := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: metav1.Time{Time: expectedTime},
		},
	}
	refVal := runAndCheckExpressionWithData(t, "SinceSecond(node)", map[string]any{
		"node": n,
	}, types.DoubleType)

	actual := refVal.Value().(float64)
	if actual < 0 {
		t.Errorf("expected positive value, got %v", actual)
	}
}

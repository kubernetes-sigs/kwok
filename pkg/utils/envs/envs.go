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

package envs

import (
	"os"

	"sigs.k8s.io/kwok/pkg/utils/format"
)

var (
	// EnvPrefix is the key prefix of the environment variable value
	EnvPrefix = "KWOK_"
)

// GetEnv returns the value of the environment variable named by the key.
func GetEnv[T any](key string, def T) T {
	value, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	if value == "" {
		return def
	}
	t, err := format.Parse[T](value)
	if err != nil {
		return def
	}
	return t
}

// GetEnvWithPrefix returns the value of the environment variable named by the key with kwok prefix.
func GetEnvWithPrefix[T any](key string, def T) T {
	return GetEnv(EnvPrefix+key, def)
}

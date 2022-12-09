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
	"strconv"
)

var (
	// EnvPrefix is the key prefix of the environment variable value
	EnvPrefix = "KWOK_"
)

// GetEnv returns the value of the environment variable named by the key.
func GetEnv(key, def string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	return value
}

// GetEnvWithPrefix returns the value of the environment variable named by the key with kwok prefix.
func GetEnvWithPrefix(key, def string) string {
	return GetEnv(EnvPrefix+key, def)
}

// GetEnvUint32WithPrefix returns the value of the environment variable named by the key.
func GetEnvUint32WithPrefix(key string, def uint32) uint32 {
	v := GetEnvWithPrefix(key, "")
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return uint32(i)
}

// GetEnvBoolWithPrefix returns the value of the environment variable named by the key.
func GetEnvBoolWithPrefix(key string, def bool) bool {
	v := GetEnvWithPrefix(key, "")
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

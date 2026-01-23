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

package log

// The following is Level security definitions.
const (
	InfoLevelSecurity  = "info"
	DebugLevelSecurity = "debug"
	WarnLevelSecurity  = "warn"
	ErrorLevelSecurity = "error"
)

// ToKlogLevel maps the current logging level to a Klog level integer
func ToKlogLevel(level Level) int {
	if int(level) > 0 {
		return 0
	}
	return -int(level)
}

// ToLogSeverityLevel maps the current logging level to a severity level string
func ToLogSeverityLevel(level Level) string {
	switch {
	case level < LevelInfo:
		return DebugLevelSecurity
	case level < LevelWarn:
		return InfoLevelSecurity
	case level < LevelError:
		return WarnLevelSecurity
	default:
		return ErrorLevelSecurity
	}
}

// ToZapLevel maps the current logging level to a Zap level string
func ToZapLevel(level Level) string {
	switch {
	case level < LevelInfo:
		return DebugLevelSecurity
	case level < LevelWarn:
		return InfoLevelSecurity
	default:
		return ErrorLevelSecurity
	}
}

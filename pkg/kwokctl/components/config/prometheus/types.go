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

// Package prometheus copy from https://github.com/prometheus/prometheus/blob/919648cafc2c07ed5c1d5dd657b8080bee331aaf/config/config.go#L243
package prometheus

type GlobalConfig struct {
	ScrapeInterval     string `json:"scrape_interval,omitempty"`
	ScrapeTimeout      string `json:"scrape_timeout,omitempty"`
	EvaluationInterval string `json:"evaluation_interval,omitempty"`
}

type MetricsDiscovery struct {
	Scheme string `json:"scheme,omitempty"`
	Host   string `json:"host,omitempty"`
	Path   string `json:"path,omitempty"`
}

type Metric struct {
	Scheme             string `json:"scheme,omitempty"`
	Path               string `json:"metrics_path,omitempty"`
	CertPath           string `json:"cert_file,omitempty"`
	KeyPath            string `json:"key_file,omitempty"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	Host               string `json:"host,omitempty"`
}

type ScrapeConfig struct {
	JobName         string         `json:"job_name"`
	HttpSdConfigs   []HTTPSDConfig `json:"http_sd_configs,omitempty"`
	Scheme          string         `json:"scheme,omitempty"`
	HonorTimestamps bool           `json:"honor_timestamps,omitempty"`
	MetricsPath     string         `json:"metrics_path,omitempty"`
	FollowRedirects bool           `json:"follow_redirects,omitempty"`
	EnableHttp2     bool           `json:"enable_http2,omitempty"`
	TLSConfig       *TLSConfig     `json:"tls_config,omitempty"`
	StaticConfigs   []StaticConfig `json:"static_configs,omitempty"`
}

type HTTPSDConfig struct {
	URL string `json:"url"`
}

type TLSConfig struct {
	CertFile           string `json:"cert_file"`
	KeyFile            string `json:"key_file"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
}

type StaticConfig struct {
	Targets []string `json:"targets"`
}

// Config is the top-level configuration for Prometheus's config files.
type Config struct {
	GlobalConfig  GlobalConfig    `json:"global"`
	ScrapeConfigs []*ScrapeConfig `json:"scrape_configs,omitempty"`
}

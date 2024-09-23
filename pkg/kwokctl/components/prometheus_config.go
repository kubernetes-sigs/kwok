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

package components

import (
	"fmt"
	"net/url"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/kwokctl/components/config/prometheus"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

// BuildPrometheus builds the Prometheus configuration.
func BuildPrometheus(conf BuildPrometheusConfig) (string, error) {
	config := prometheus.Config{
		GlobalConfig: prometheus.GlobalConfig{
			ScrapeInterval:     "15s",
			ScrapeTimeout:      "10s",
			EvaluationInterval: "15s",
		},
		ScrapeConfigs: convertToScrapeConfigs(conf.Components),
	}

	configJSON, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("build prometheus config error: %w", err)
	}
	return string(configJSON), nil
}

// convertToScrapeConfigs converts internalversion.Component to prometheus.ScrapeConfig.
func convertToScrapeConfigs(components []internalversion.Component) []*prometheus.ScrapeConfig {
	var scrapeConfigs []*prometheus.ScrapeConfig
	for _, c := range components {
		if md := c.MetricsDiscovery; md != nil {
			scrapeConfig := &prometheus.ScrapeConfig{}
			scrapeConfig.JobName = fmt.Sprintf("%s-metrics-discovery", c.Name)
			u := url.URL{
				Scheme: md.Scheme,
				Host:   md.Host,
				Path:   md.Path,
			}
			scrapeConfig.HttpSdConfigs = []prometheus.HTTPSDConfig{
				{
					URL: u.String(),
				},
			}
			scrapeConfigs = append(scrapeConfigs, scrapeConfig)
		}

		if m := c.Metric; m != nil {
			scrapeConfig := &prometheus.ScrapeConfig{}
			scrapeConfig.JobName = c.Name
			scrapeConfig.Scheme = m.Scheme
			scrapeConfig.MetricsPath = m.Path
			scrapeConfig.HonorTimestamps = true
			scrapeConfig.EnableHttp2 = true
			scrapeConfig.FollowRedirects = true

			if scrapeConfig.Scheme == "https" {
				scrapeConfig.TLSConfig = &prometheus.TLSConfig{}
				scrapeConfig.TLSConfig.CertFile = m.CertPath
				scrapeConfig.TLSConfig.KeyFile = m.KeyPath
				scrapeConfig.TLSConfig.InsecureSkipVerify = true
			}

			scrapeConfig.StaticConfigs = []prometheus.StaticConfig{
				{
					Targets: []string{
						m.Host,
					},
				},
			}
			scrapeConfigs = append(scrapeConfigs, scrapeConfig)
		}
	}
	return scrapeConfigs
}

// BuildPrometheusConfig is the configuration for building the prometheus config
type BuildPrometheusConfig struct {
	Components []internalversion.Component
}

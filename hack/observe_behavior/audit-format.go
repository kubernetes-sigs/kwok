/*
Copyright 2025 The Kubernetes Authors.

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

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	auditv1 "k8s.io/apiserver/pkg/apis/audit/v1"
	utilsslices "sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/yaml"
)

func main() {

	input := os.Stdin
	output := os.Stdout

	fmt.Fprintln(output, "# Formatted Audit Log")

	scanner := bufio.NewScanner(input)
	var event auditv1.Event
	for scanner.Scan() {
		line := scanner.Bytes()
		err := json.Unmarshal(line, &event)
		if err != nil {
			fmt.Fprintf(output, "Error parsing JSON: %v\n", err)
			os.Exit(1)
			return
		}

		if event.RequestObject.ContentType != "application/json" {
			fmt.Fprintf(output, "---\n# Skipping resource with unsupported content type: %s\n", event.RequestObject.ContentType)
			continue
		}
		yamlData, err := yaml.JSONToYAML(event.RequestObject.Raw)
		if err != nil {
			fmt.Fprintf(output, "---\n# Error converting to YAML: %v\n", err)
			continue
		}

		timestamp := event.RequestReceivedTimestamp.UTC().Format(time.RFC3339)
		operation := strings.SplitN(event.UserAgent, "/", 2)[0]
		verb := event.Verb
		requestURI := event.RequestURI

		info := Info{
			Timestamp:  timestamp,
			Operator:   operation,
			Verb:       verb,
			RequestURI: requestURI,
			ObjectRef: ObjectRef{
				Resource:    event.ObjectRef.Resource,
				Subresource: event.ObjectRef.Subresource,
				Name:        event.ObjectRef.Name,
				Namespace:   event.ObjectRef.Namespace,
			},
		}
		infoYAML, err := yaml.Marshal(info)
		if err != nil {
			fmt.Printf("# Error marshaling info to YAML: %v\n", err)
			continue
		}

		fmt.Fprintln(output, "---")
		output.Write(commentsYAML(infoYAML))
		output.Write(yamlData)
	}

	fmt.Fprintln(output, "---")
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(output, "Error reading file: %v\n", err)
		os.Exit(1)
	}
}

func commentsYAML(data []byte) []byte {
	return bytes.Join(utilsslices.Map(bytes.Split(data, []byte{'\n'}), func(line []byte) []byte {
		if len(line) == 0 {
			return []byte{}
		}
		return append([]byte("# "), line...)
	}), []byte{'\n'})
}

type Info struct {
	Timestamp  string    `json:"timestamp"`
	Operator   string    `json:"operator"`
	Verb       string    `json:"verb"`
	RequestURI string    `json:"requestURI"`
	ObjectRef  ObjectRef `json:"objectRef"`
}

type ObjectRef struct {
	Resource    string `json:"resource"`
	Subresource string `json:"subresource,omitempty"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
}

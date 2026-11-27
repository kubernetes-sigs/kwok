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

package compose

import (
	"errors"
	"strings"
	"testing"
)

// The inspect results are trimmed to the fields kwokctl reads, with their real shapes.
func TestInspectNetworkAddress(t *testing.T) {
	const network = "kwok-test"
	tests := []struct {
		name    string
		inspect string
		want    string
	}{
		{
			name: "docker",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"com.docker.compose.project":"kwok-test"}},
				"NetworkSettings":{"IPAddress":"","Networks":{"kwok-test":{"IPAddress":"172.18.0.3","GlobalIPv6Address":""}}}}]`,
			want: "172.18.0.3",
		},
		{
			name: "podman",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"io.podman.compose.project":"kwok-test"}},
				"NetworkSettings":{"IPAddress":"","Networks":{"kwok-test":{"IPAddress":"10.89.0.4","GlobalIPv6Address":"","NetworkID":"kwok-test"}}}}]`,
			want: "10.89.0.4",
		},
		{
			name: "nerdctl since v2.4",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"kwok-test\"]"}},
				"NetworkSettings":{"IPAddress":"10.4.0.7","Networks":{"kwok-test":{"IPAddress":"10.4.0.7"}}}}]`,
			want: "10.4.0.7",
		},
		{
			name: "nerdctl before v2.4",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"kwok-test\"]"}},
				"NetworkSettings":{"IPAddress":"10.4.0.7","Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"}}}}]`,
			want: "10.4.0.7",
		},
		{
			name: "nerdctl before v2.4 on the second network",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"bridge\",\"kwok-test\"]"}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"},"unknown-eth1":{"IPAddress":"10.5.0.2"}}}}]`,
			want: "10.5.0.2",
		},
		{
			name: "nerdctl before v2.4 not on the network",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"bridge\"]"}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"}}}}]`,
			want: "",
		},
		{
			name: "nerdctl without the networks label",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"}}}}]`,
			want: "",
		},
		{
			name: "invalid networks label",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"kwok-test"}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"}}}}]`,
			want: "",
		},
		{
			name: "another network only",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.17.0.2"}}}}]`,
			want: "",
		},
		{
			name: "several networks",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"bridge":{"IPAddress":"172.17.0.2"},"kwok-test":{"IPAddress":"172.18.0.3"}}}}]`,
			want: "172.18.0.3",
		},
		{
			name: "ipv6 only",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"","GlobalIPv6Address":"fd00:18::3"}}}}]`,
			want: "fd00:18::3",
		},
		{
			name: "dual stack prefers ipv4",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"172.18.0.3","GlobalIPv6Address":"fd00:18::3"}}}}]`,
			want: "172.18.0.3",
		},
		{
			name: "invalid address",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"172.18.0","GlobalIPv6Address":"not-an-ip"}}}}]`,
			want: "",
		},
		{
			name: "unspecified address",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"0.0.0.0","GlobalIPv6Address":"::"}}}}]`,
			want: "",
		},
		{
			name: "ipv4 in the ipv6 field",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"GlobalIPv6Address":"172.18.0.3"}}}}]`,
			want: "",
		},
		{
			name: "nerdctl before v2.4 with networks in another order",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"kwok-test\",\"bridge\"]"}},
				"NetworkSettings":{"Networks":{"unknown-eth1":{"IPAddress":"10.4.0.7"},"unknown-eth0":{"IPAddress":"10.5.0.2"}}}}]`,
			want: "10.5.0.2",
		},
		{
			// The interface of the network is missing, and the other one belongs to another network.
			name: "nerdctl before v2.4 without the interface of the network",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"bridge\",\"kwok-test\"]"}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"}}}}]`,
			want: "",
		},
		{
			// Only "unknown-eth<i>", as go-cni names the i-th network, is taken.
			name: "nerdctl before v2.4 with other interface names",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"kwok-test\"]"}},
				"NetworkSettings":{"Networks":{"unknown-net0":{"IPAddress":"10.4.0.7"},"unknown-eth0.100":{"IPAddress":"10.5.0.2"},
				"unknown-eth00":{"IPAddress":"10.6.0.2"},"unknown-eth1":{"IPAddress":"10.7.0.2"}}}}]`,
			want: "",
		},
		{
			name: "nerdctl before v2.4 with a duplicated network",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"kwok-test\",\"kwok-test\"]"}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"},"unknown-eth1":{"IPAddress":"10.4.0.8"}}}}]`,
			want: "10.4.0.7",
		},
		{
			name: "name preferred over the interface",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"bridge\",\"kwok-test\"]"}},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"10.5.0.2"},"unknown-eth1":{"IPAddress":"10.9.0.9"}}}}]`,
			want: "10.5.0.2",
		},
		{
			name: "nerdctl before v2.4 with an empty networks label",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[]"}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"10.4.0.7"}}}}]`,
			want: "",
		},
		{
			name: "nerdctl before v2.4 with ipv6 only",
			inspect: `[{"State":{"Running":true},"Config":{"Labels":{"nerdctl/networks":"[\"kwok-test\"]"}},
				"NetworkSettings":{"Networks":{"unknown-eth0":{"IPAddress":"","GlobalIPv6Address":"fd00:4::7"}}}}]`,
			want: "fd00:4::7",
		},
		{
			name: "link-local and loopback addresses",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"169.254.0.3","GlobalIPv6Address":"fe80::3"}}}}]`,
			want: "",
		},
		{
			name: "loopback addresses",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"127.0.0.1","GlobalIPv6Address":"::1"}}}}]`,
			want: "",
		},
		{
			name: "invalid ipv4 with a valid ipv6",
			inspect: `[{"State":{"Running":true},
				"NetworkSettings":{"Networks":{"kwok-test":{"IPAddress":"172.18.0.300","GlobalIPv6Address":"fd00:18::3"}}}}]`,
			want: "fd00:18::3",
		},
		{
			name:    "not connected",
			inspect: `[{"State":{"Running":true},"NetworkSettings":{"Networks":{}}}]`,
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := parseInspect([]byte(tt.inspect))
			if err != nil {
				t.Fatal(err)
			}
			if got := status.networkAddress(network); got != tt.want {
				t.Errorf("networkAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProbeHost(t *testing.T) {
	address := &inspectStatus{}
	address.NetworkSettings.Networks = map[string]inspectEndpoint{"kwok-test": {IPAddress: "172.18.0.3"}}
	none := &inspectStatus{}

	tests := []struct {
		name    string
		cluster string
		status  *inspectStatus
		want    string
		wantErr error
	}{
		{name: "address", cluster: "test", status: address, want: "172.18.0.3"},
		{name: "unknown status", cluster: "test", status: nil, want: "kwok-test-jobset"},
		{name: "short name", cluster: "test", status: none, want: "kwok-test-jobset"},
		{
			// "kwok-" + cluster + "-jobset" is 63 characters, the longest a DNS label can be.
			name:    "longest name",
			cluster: strings.Repeat("a", 51),
			status:  none,
			want:    "kwok-" + strings.Repeat("a", 51) + "-jobset",
		},
		{
			// The container name is 64 characters, longer than a DNS label can be.
			name:    "long name",
			cluster: strings.Repeat("a", 52),
			status:  none,
			wantErr: errProbeAddressUnavailable,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFakeComposeWithName(t, tt.cluster)
			if tt.status == address {
				// The fixture names the network after the "test" cluster.
				tt.status.NetworkSettings.Networks = map[string]inspectEndpoint{c.Name(): {IPAddress: "172.18.0.3"}}
			}
			got, err := c.probeHost(tt.status, "jobset")
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Errorf("probeHost() = %q, %v, want %q, %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

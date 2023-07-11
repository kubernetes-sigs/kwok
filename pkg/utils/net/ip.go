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

package net

import (
	"encoding/binary"
	"net"
)

// GetAllIPs returns all IPs of the host.
func GetAllIPs() ([]string, error) {
	iface, err := net.Interfaces()
	if err != nil {
		return []string{}, err
	}
	var ips []string
	for _, i := range iface {
		if i.Flags&net.FlagUp == 0 || i.Flags&net.FlagRunning == 0 {
			continue
		}

		cidrs, err := i.Addrs()
		if err != nil {
			return []string{}, err
		}
		for _, addr := range cidrs {
			if ip, ok := addr.(*net.IPNet); ok {
				ips = append(ips, ip.IP.String())
			}
		}
	}
	return ips, nil
}

// AddIP adds or subtracts the IP.
func AddIP(ip net.IP, add uint64) net.IP {
	if len(ip) < 8 || add == 0 {
		return ip
	}

	out := make(net.IP, len(ip))
	copy(out, ip)

	i := binary.BigEndian.Uint64(out[len(out)-8:])
	i += add

	binary.BigEndian.PutUint64(out[len(out)-8:], i)
	return out
}

// ParseCIDR parses a CIDR string.
func ParseCIDR(s string) (*net.IPNet, error) {
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return nil, err
	}
	ipnet.IP = ip
	return ipnet, nil
}

// AddCIDR adds the CIDR.
func AddCIDR(cidr string, index int) (string, error) {
	ipnet, err := ParseCIDR(cidr)
	if err != nil {
		return "", err
	}
	ones, bits := ipnet.Mask.Size()
	ipnet.IP = AddIP(ipnet.IP, uint64((1<<(bits-ones))*index))
	return ipnet.String(), nil
}

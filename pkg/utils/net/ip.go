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
	"fmt"
	"math/big"
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

// AddIPStr adds the IP string with the given offset.
func AddIPStr(ipStr string, index int) (string, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "", fmt.Errorf("invalid ip %q", ipStr)
	}
	return AddIP(ip, index).String(), nil
}

// AddIP adds or subtracts the IP.
func AddIP(ip net.IP, index int) net.IP {
	if len(ip) < 8 || index == 0 {
		return ip
	}

	out := make(net.IP, len(ip))
	copy(out, ip)

	low := binary.BigEndian.Uint64(out[len(out)-8:])
	if index < 0 {
		low -= uint64(-index)
	} else {
		low += uint64(index)
	}
	binary.BigEndian.PutUint64(out[len(out)-8:], low)
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

// AddCIDRStr adds the CIDR.
func AddCIDRStr(cidr string, index int) (string, error) {
	ipnet, err := ParseCIDR(cidr)
	if err != nil {
		return "", err
	}

	ipnet, err = AddCIDR(ipnet, index)
	if err != nil {
		return "", err
	}
	return ipnet.String(), nil
}

// AddCIDRStr adds the CIDR.
func AddCIDR(ipnet *net.IPNet, index int) (*net.IPNet, error) {
	ones, bits := ipnet.Mask.Size()
	ip := ipnet.IP
	if bits == net.IPv4len*8 {
		ip = ip.To4()
	} else {
		ip = ip.To16()
	}
	if ip == nil {
		return nil, fmt.Errorf("invalid cidr %q", ipnet.String())
	}

	offset := new(big.Int).SetInt64(int64(index))
	offset.Lsh(offset, uint(bits-ones))

	ipInt := new(big.Int).SetBytes(ip)
	ipInt.Add(ipInt, offset)

	maxIP := new(big.Int).Lsh(big.NewInt(1), uint(bits))
	if ipInt.Cmp(maxIP) >= 0 {
		return nil, fmt.Errorf("cidr %q with index %d overflows ip range", ipnet.String(), index)
	}

	return &net.IPNet{
		IP:   ipInt.FillBytes(make([]byte, len(ip))),
		Mask: ipnet.Mask,
	}, nil
}

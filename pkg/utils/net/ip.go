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
	"fmt"
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

// ParseIP parses a IP string.
func ParseIP(s string) (net.IP, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, fmt.Errorf("invalid ip %q", s)
	}
	return ip, nil
}

// AddIPStr adds the IP string with the given offset.
func AddIPStr(ipStr string, index int) (string, error) {
	ip, err := ParseIP(ipStr)
	if err != nil {
		return "", err
	}
	return AddIP(ip, index).String(), nil
}

// AddIP adds the IP with the given offset.
func AddIP(ip net.IP, index int) net.IP {
	if index == 0 {
		return ip
	}

	return addBytesInt(ip, int64(index))
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

// AddCIDRStr adds the CIDR string with the given offset.
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

// AddCIDR adds the CIDR with the given offset.
func AddCIDR(ipnet *net.IPNet, index int) (*net.IPNet, error) {
	ones, bits := ipnet.Mask.Size()
	hostBits := bits - ones
	return &net.IPNet{
		IP:   addBytes(ipnet.IP, hostBits, int64(index)),
		Mask: ipnet.Mask,
	}, nil
}

// addBytes adds the offset (as a byte slice) to the IP byte slice.
func addBytes(b []byte, hostBits int, index int64) []byte {
	if len(b) == 0 || index == 0 {
		return b
	}

	// Use fast path with addBytesInt when result fits in int64 (up to 2^63-1)
	offset := index << hostBits
	if offset > 0 && offset <= 1<<62-1 {
		return addBytesInt(b, offset)
	}

	offsetBytes := leftShiftInt(index, hostBits)

	out := make([]byte, len(b))
	copy(out, b)

	// Add offset to the byte slice from right to left
	carry := 0
	for i := len(out) - 1; i >= 0; i-- {
		// Get the corresponding byte from offset (aligned to the right)
		offsetIdx := len(offsetBytes) - (len(out) - i)
		var offset byte
		if offsetIdx >= 0 && offsetIdx < len(offsetBytes) {
			offset = offsetBytes[offsetIdx]
		}

		sum := int(out[i]) + int(offset) + carry
		out[i] = byte(sum % 256)
		carry = sum / 256
	}

	return out
}

// leftShiftInt converts an integer to a byte slice representing
// the value left-shifted by hostBits positions.
func leftShiftInt(value int64, hostBits int) []byte {
	// Convert value to bytes
	var bytes []byte
	if value == 0 {
		return []byte{0}
	}
	tmp := value
	for tmp > 0 {
		bytes = append(bytes, byte(tmp&0xFF))
		tmp >>= 8
	}
	// Reverse to get big-endian
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	// Now we need to shift left by hostBits bits
	// This means we need to add hostBits/8 bytes of zeros at the end
	// and handle the remaining bits
	byteShift := hostBits / 8
	bitShift := hostBits % 8

	if byteShift > 0 {
		// Add zeros at the end
		bytes = append(bytes, make([]byte, byteShift)...)
	}

	if bitShift > 0 {
		// Shift each byte left by bitShift bits
		// and add the high bits to the next byte
		carry := 0
		for i := len(bytes) - 1; i >= 0; i-- {
			newCarry := int(bytes[i] >> (8 - bitShift))
			bytes[i] = byte((int(bytes[i]) << bitShift) | carry)
			carry = newCarry
		}
		if carry > 0 {
			bytes = append([]byte{byte(carry)}, bytes...)
		}
	}

	return bytes
}

// addBytesInt is a helper function to add an integer offset to a byte slice representing an IP address.
func addBytesInt(b []byte, index int64) []byte {
	if len(b) == 0 || index == 0 {
		return b
	}

	out := make([]byte, len(b))
	copy(out, b)

	if index > 0 {
		carry := index
		for i := len(out) - 1; i >= 0 && carry > 0; i-- {
			sum := int64(out[i]) + (carry & 0xFF)
			out[i] = byte(sum & 0xFF)
			carry = (carry >> 8) + (sum >> 8)
		}
	} else {
		carry := -index
		for i := len(out) - 1; i >= 0 && carry > 0; i-- {
			diff := int64(out[i]) - (carry & 0xFF)
			if diff < 0 {
				out[i] = byte((diff + 256) & 0xFF)
				carry = (carry >> 8) + 1
			} else {
				out[i] = byte(diff & 0xFF)
				carry >>= 8
			}
		}
	}

	return out
}

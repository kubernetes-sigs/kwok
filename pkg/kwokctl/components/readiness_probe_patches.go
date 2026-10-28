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

package components

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/consts"
)

// ErrReadinessProbeConflict is returned when the arguments of a component keep its readiness probe from succeeding.
var ErrReadinessProbeConflict = errors.New("readiness probe cannot succeed with the component arguments")

// probeFlags names the flags of a component that its readiness probe depends on.
type probeFlags struct {
	// port sets the port the probe connects to.
	port string
	// bind sets the address that port listens on.
	bind string
	// listen sets both, as host:port.
	listen string
	// urls sets both, as a comma-separated list of URLs.
	urls string
	// check reports why the probe cannot be answered with the given flags.
	check func(flags map[string]string, probe *internalversion.ComponentProbe) error
}

// readinessProbeFlags returns the flags that the readiness probe of a component depends on.
// The probe of kube-apiserver, kube-controller-manager and kube-scheduler targets the secure port
// when it is HTTPS, and the insecure port otherwise.
func readinessProbeFlags(component *internalversion.Component) (probeFlags, bool) {
	https := component.ReadinessProbe.HTTPGet.Scheme == internalversion.URISchemeHTTPS
	switch component.Name {
	case consts.ComponentKubeApiserver:
		if https {
			return probeFlags{port: "secure-port", bind: "bind-address", check: checkKubeApiserverAnonymous}, true
		}
		return probeFlags{port: "insecure-port", bind: "insecure-bind-address"}, true
	case consts.ComponentKubeControllerManager, consts.ComponentKubeScheduler:
		if https {
			return probeFlags{port: "secure-port", bind: "bind-address", check: checkAlwaysAllowPaths}, true
		}
		return probeFlags{port: "port", bind: "address"}, true
	case consts.ComponentKubeApiserverInsecureProxy:
		return probeFlags{port: "port", bind: "address"}, true
	case consts.ComponentKwokController:
		return probeFlags{listen: "server-address"}, true
	case consts.ComponentPrometheus:
		return probeFlags{listen: "web.listen-address"}, true
	case consts.ComponentEtcd:
		return probeFlags{urls: "listen-client-urls"}, true
	case consts.ComponentLWS, consts.ComponentNodeReadinessController:
		return probeFlags{listen: "health-probe-bind-address"}, true
	case consts.ComponentSchedulerPlugins:
		return probeFlags{listen: "probeAddr"}, true
	}
	return probeFlags{}, false
}

// checkKubeApiserverAnonymous reports whether anonymous requests, which the probe sends, are rejected.
// kube-apiserver turns anonymous authentication off with the AlwaysAllow authorizer.
func checkKubeApiserverAnonymous(flags map[string]string, _ *internalversion.ComponentProbe) error {
	if v, ok := flags["anonymous-auth"]; ok {
		if allow, err := strconv.ParseBool(v); err != nil || !allow {
			return fmt.Errorf("--anonymous-auth=%s rejects the unauthenticated readiness probe", v)
		}
	}
	if v, ok := flags["authorization-mode"]; ok && slices.Contains(strings.Split(v, ","), "AlwaysAllow") {
		return fmt.Errorf("--authorization-mode=%s turns off anonymous authentication, which rejects the unauthenticated readiness probe", v)
	}
	return nil
}

// checkAlwaysAllowPaths reports whether the probed path no longer skips authorization.
func checkAlwaysAllowPaths(flags map[string]string, probe *internalversion.ComponentProbe) error {
	v, ok := flags["authorization-always-allow-paths"]
	if !ok {
		return nil
	}
	if !alwaysAllowed(v, probe.HTTPGet.Path) {
		return fmt.Errorf("--authorization-always-allow-paths=%s does not allow the unauthenticated readiness probe of %s", v, probe.HTTPGet.Path)
	}
	return nil
}

// alwaysAllowed reports whether path matches the list of --authorization-always-allow-paths,
// where each entry matches exactly, or by prefix if it ends in "*", and the leading "/" is optional.
func alwaysAllowed(paths, path string) bool {
	path = strings.TrimPrefix(path, "/")
	for p := range strings.SplitSeq(paths, ",") {
		p = strings.TrimPrefix(p, "/")
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else if p == path {
			return true
		}
	}
	return false
}

// ReconcileReadinessProbe updates the readiness probe of a component after its arguments
// were patched from original to component.Args, so that it follows the port the component listens on.
// The probe is never removed: if the patched arguments keep it from succeeding,
// the component stays not ready, and the returned error says why.
func ReconcileReadinessProbe(component *internalversion.Component, original []string) error {
	if component.ReadinessProbe == nil || component.ReadinessProbe.HTTPGet == nil || slices.Equal(original, component.Args) {
		return nil
	}
	pf, ok := readinessProbeFlags(component)
	if !ok {
		return nil
	}
	before := parseFlags(original)
	after := parseFlags(component.Args)
	changed := func(flag string) (string, bool) {
		if flag == "" {
			return "", false
		}
		v, ok := after[flag]
		return v, ok && v != before[flag]
	}

	var errs []error
	if v, ok := changed(pf.port); ok {
		if p, err := parseProbePort(v); err == nil {
			component.ReadinessProbe.HTTPGet.Port = p
		}
	}
	if v, ok := changed(pf.listen); ok {
		if host, p, err := net.SplitHostPort(v); err == nil {
			if p, err := parseProbePort(p); err == nil {
				component.ReadinessProbe.HTTPGet.Port = p
			}
			if !listensOnAllInterfaces(host) {
				errs = append(errs, unreachableAddress(pf.listen, v))
			}
		}
	}
	if v, ok := changed(pf.urls); ok {
		if u, ok := probedURL(v); ok {
			p, _ := parseProbePort(u.Port())
			component.ReadinessProbe.HTTPGet.Port = p
		}
	}
	if v, ok := changed(pf.bind); ok && !listensOnAllInterfaces(v) {
		errs = append(errs, unreachableAddress(pf.bind, v))
	}
	if err := ReadinessProbeConflict(*component); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// ReadinessProbeConflict returns an error wrapping ErrReadinessProbeConflict
// if the arguments of a component keep its readiness probe from succeeding.
func ReadinessProbeConflict(component internalversion.Component) error {
	if component.ReadinessProbe == nil || component.ReadinessProbe.HTTPGet == nil {
		return nil
	}
	pf, ok := readinessProbeFlags(&component)
	if !ok {
		return nil
	}
	flags := parseFlags(component.Args)
	port := component.ReadinessProbe.HTTPGet.Port

	var err error
	if v, ok := flags[pf.port]; ok && pf.port != "" {
		if p, perr := parseProbePort(v); perr != nil || p != port {
			err = fmt.Errorf("--%s=%s does not serve the readiness probe on port %d", pf.port, v, port)
		}
	}
	if v, ok := flags[pf.listen]; ok && pf.listen != "" && err == nil {
		_, p, serr := net.SplitHostPort(v)
		if pp, perr := parseProbePort(p); serr != nil || perr != nil || pp != port {
			err = fmt.Errorf("--%s=%s does not serve the readiness probe on port %d", pf.listen, v, port)
		}
	}
	if v, ok := flags[pf.urls]; ok && pf.urls != "" && err == nil {
		if !servesPlainHTTP(v, port) {
			err = fmt.Errorf("--%s=%s does not serve the plain HTTP readiness probe on port %d", pf.urls, v, port)
		}
	}
	if err == nil && pf.check != nil {
		err = pf.check(flags, component.ReadinessProbe)
	}
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrReadinessProbeConflict, component.Name, err)
	}
	return nil
}

// probedURL returns the first plain HTTP URL of a list that listens on all interfaces.
func probedURL(urls string) (*url.URL, bool) {
	for raw := range strings.SplitSeq(urls, ",") {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "http" || !listensOnAllInterfaces(u.Hostname()) {
			continue
		}
		if _, err := parseProbePort(u.Port()); err != nil {
			continue
		}
		return u, true
	}
	return nil, false
}

// servesPlainHTTP reports whether a list of URLs has a plain HTTP URL on port.
func servesPlainHTTP(urls string, port uint32) bool {
	for raw := range strings.SplitSeq(urls, ",") {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme != "http" {
			continue
		}
		if p, err := parseProbePort(u.Port()); err == nil && p == port {
			return true
		}
	}
	return false
}

// listensOnAllInterfaces reports whether a bind address listens on all interfaces,
// which the address the runtime probes is always one of.
func listensOnAllInterfaces(address string) bool {
	if address == "" {
		return true
	}
	ip := net.ParseIP(address)
	return ip != nil && ip.IsUnspecified()
}

func unreachableAddress(flag, value string) error {
	return fmt.Errorf("--%s=%s may not be reachable by the readiness probe, which then keeps failing", flag, value)
}

func parseProbePort(value string) (uint32, error) {
	p, err := strconv.ParseUint(value, 10, 16)
	if err != nil || p == 0 {
		return 0, fmt.Errorf("invalid port %q", value)
	}
	return uint32(p), nil
}

// parseFlags returns the value of each flag in args, where the last occurrence of a flag wins.
// A flag without a value is "true".
func parseFlags(args []string) map[string]string {
	flags := map[string]string{}
	for _, arg := range args {
		name, ok := strings.CutPrefix(arg, "--")
		if !ok {
			continue
		}
		key, value, hasValue := strings.Cut(name, "=")
		if !hasValue {
			value = "true"
		}
		flags[key] = value
	}
	return flags
}

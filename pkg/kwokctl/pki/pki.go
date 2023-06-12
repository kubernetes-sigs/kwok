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

package pki

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"sigs.k8s.io/kwok/pkg/utils/slices"
)

var (
	// DefaultUser is the default user for the admin user
	DefaultUser = "kwok-admin"
	// DefaultGroups is the default groups for the admin user
	DefaultGroups = []string{
		"system:masters",
	}
	// DefaultAltNames is the default alt names for the admin user
	DefaultAltNames = []string{
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc.cluster.local",
		"localhost",
		"127.0.0.1",
		"::1",
	}
)

// GeneratePki generates the pki for kwokctl
func GeneratePki(pkiPath string, sans ...string) error {
	now := time.Now()
	notBefore := now.UTC()
	notAfter := now.Add(CertificateValidity).UTC()

	// Generate CA
	caCert, caKey, err := GenerateCA("kwok-ca", notBefore, notAfter)
	if err != nil {
		return fmt.Errorf("failed to generate CA: %w", err)
	}
	err = WriteCertAndKey(pkiPath, "ca", caCert, caKey)
	if err != nil {
		return fmt.Errorf("failed to write CA: %w", err)
	}

	// Generate admin cert, use single cert for all components
	allSANs := DefaultAltNames
	if len(sans) != 0 {
		allSANs = append(allSANs, sans...)
	}
	cert, key, err := GenerateSignCert(DefaultUser, caCert, caKey, notBefore, notAfter, DefaultGroups, allSANs)
	if err != nil {
		return fmt.Errorf("failed to generate admin cert and key: %w", err)
	}
	err = WriteCertAndKey(pkiPath, "admin", cert, key)
	if err != nil {
		return fmt.Errorf("failed to write admin cert and key: %w", err)
	}
	return nil
}

// GenerateCA generates a CA certificate and key.
func GenerateCA(cn string, notBefore, notAfter time.Time) (cert *x509.Certificate, key crypto.Signer, err error) {
	return NewCertificateAuthority(CertConfig{
		CommonName:         cn,
		PublicKeyAlgorithm: x509.RSA,
		NotAfter:           notAfter,
		NotBefore:          notBefore,
	})
}

// GenerateSignCert generates a certificate and key signed by the given CA.
func GenerateSignCert(cn string, caCert *x509.Certificate, caKey crypto.Signer, notBefore, notAfter time.Time, organizations []string, sans []string) (cert *x509.Certificate, key crypto.Signer, err error) {
	alt := AltNames{}

	if len(sans) != 0 {
		sans = slices.Unique(sans)
		for _, name := range sans {
			ip := net.ParseIP(name)
			if ip != nil {
				alt.IPs = append(alt.IPs, ip)
			} else {
				alt.DNSNames = append(alt.DNSNames, name)
			}
		}
	}

	certConfig := CertConfig{
		CommonName:         cn,
		Organization:       organizations,
		Usages:             []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		AltNames:           alt,
		PublicKeyAlgorithm: x509.RSA,
		NotAfter:           notAfter,
		NotBefore:          notBefore,
	}
	return NewCertAndKey(caCert, caKey, certConfig)
}

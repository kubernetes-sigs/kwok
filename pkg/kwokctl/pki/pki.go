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
	"net"
	"time"
)

type pkiSuite struct {
	cert   *x509.Certificate
	key    crypto.Signer
	caCert *x509.Certificate
	caKey  crypto.Signer
}

func generatePki() (*pkiSuite, error) {
	now := time.Now()
	notBefore := now.UTC()
	notAfter := now.Add(CertificateValidity).UTC()
	caCert, caKey, err := NewCertificateAuthority(CertConfig{
		CommonName:         "kwok-ca",
		PublicKeyAlgorithm: x509.RSA,
		NotAfter:           notAfter,
		NotBefore:          notBefore,
	})
	if err != nil {
		return nil, err
	}

	cert, key, err := NewCertAndKey(caCert, caKey, CertConfig{
		CommonName:   "kwok-admin",
		Organization: []string{"system:masters"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
		AltNames: AltNames{
			DNSNames: []string{
				"kubernetes",
				"kubernetes.default",
				"kubernetes.default.svc",
				"kubernetes.default.svc.cluster.local",
			},
			IPs: []net.IP{
				net.IPv4(127, 0, 0, 1),
			},
		},
		PublicKeyAlgorithm: x509.RSA,
		NotAfter:           notAfter,
		NotBefore:          notBefore,
	})
	if err != nil {
		return nil, err
	}
	return &pkiSuite{
		cert:   cert,
		key:    key,
		caCert: caCert,
		caKey:  caKey,
	}, nil
}

// GeneratePki generates a new PKI suite for the cluster.
func GeneratePki(dir string) error {
	p, err := generatePki()
	if err != nil {
		return err
	}
	err = writeCertAndKey(dir, "admin", p.cert, p.key)
	if err != nil {
		return err
	}
	err = writeCert(dir, "ca", p.caCert)
	if err != nil {
		return err
	}
	return nil
}

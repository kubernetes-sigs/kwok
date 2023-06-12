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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"sigs.k8s.io/kwok/pkg/utils/path"
)

const (
	// CertificateBlockType is a possible value for pem.Block.Type.
	CertificateBlockType = "CERTIFICATE"
	// ECPrivateKeyBlockType is a possible value for pem.Block.Type.
	ECPrivateKeyBlockType = "EC PRIVATE KEY"
	// RSAPrivateKeyBlockType is a possible value for pem.Block.Type.
	RSAPrivateKeyBlockType = "RSA PRIVATE KEY"

	// CertificateValidity is the validity period of a certificate.
	CertificateValidity = 100 * 365 * 24 * time.Hour

	rsaKeySize = 2048
)

// AltNames contains the domain names and IP addresses that will be added
// to the API Server's x509 certificate SubAltNames field. The values will
// be passed directly to the x509.Certificate object.
type AltNames struct {
	DNSNames []string
	IPs      []net.IP
}

// CertConfig contains the basic fields required for creating a certificate
type CertConfig struct {
	CommonName         string
	Organization       []string
	AltNames           AltNames
	Usages             []x509.ExtKeyUsage
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
	NotBefore          time.Time
	NotAfter           time.Time
}

// NewCertificateAuthority creates new certificate and private key for the certificate authority
func NewCertificateAuthority(config CertConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := newPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating CA certificate: %w", err)
	}

	cert, err := NewSelfSignedCACert(config, key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create self-signed CA certificate: %w", err)
	}

	return cert, key, nil
}

// NewSelfSignedCACert creates a CA certificate
func NewSelfSignedCACert(cfg CertConfig, key crypto.Signer) (*x509.Certificate, error) {
	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		NotBefore:             cfg.NotBefore,
		NotAfter:              cfg.NotAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDERBytes, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// NewIntermediateCertificateAuthority creates new certificate and private key for an intermediate certificate authority
func NewIntermediateCertificateAuthority(parentCert *x509.Certificate, parentKey crypto.Signer, config CertConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := newPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating intermediate CA certificate: %w", err)
	}

	cert, err := NewSignedCert(config, key, parentCert, parentKey, true)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign intermediate CA certificate: %w", err)
	}

	return cert, key, nil
}

// NewCertAndKey creates new certificate and key by passing the certificate authority certificate and key
func NewCertAndKey(caCert *x509.Certificate, caKey crypto.Signer, config CertConfig) (*x509.Certificate, crypto.Signer, error) {
	if len(config.Usages) == 0 {
		return nil, nil, fmt.Errorf("must specify at least one ExtKeyUsage")
	}

	key, err := newPrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key: %w", err)
	}

	cert, err := NewSignedCert(config, key, caCert, caKey, false)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign certificate: %w", err)
	}

	return cert, key, nil
}

// newPrivateKey returns a new private key.
func newPrivateKey(keyType x509.PublicKeyAlgorithm) (crypto.Signer, error) {
	switch keyType {
	case x509.ECDSA:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case x509.RSA:
		return rsa.GenerateKey(rand.Reader, rsaKeySize)
	default:
		return nil, fmt.Errorf("unsupported key type")
	}
}

// NewSignedCert creates a signed certificate using the given CA certificate and key
func NewSignedCert(cfg CertConfig, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer, isCA bool) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, fmt.Errorf("must specify a CommonName")
	}

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if isCA {
		keyUsage |= x509.KeyUsageCertSign
	}

	removeDuplicateAltNames(&cfg.AltNames)

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:              cfg.AltNames.DNSNames,
		IPAddresses:           cfg.AltNames.IPs,
		SerialNumber:          serial,
		NotBefore:             cfg.NotBefore,
		NotAfter:              cfg.NotAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           cfg.Usages,
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// removeDuplicateAltNames removes duplicate items in altNames.
func removeDuplicateAltNames(altNames *AltNames) {
	if altNames == nil {
		return
	}

	ipsKeys := make(map[string]struct{})
	var ips []net.IP
	for _, one := range altNames.IPs {
		if _, ok := ipsKeys[one.String()]; !ok {
			ipsKeys[one.String()] = struct{}{}
			ips = append(ips, one)
		}
	}
	altNames.IPs = ips
}

// ReadCertAndKey reads certificate and key from the specified location
func ReadCertAndKey(pkiPath string, name string) (*x509.Certificate, crypto.Signer, error) {
	cert, err := readCert(pkiPath, name)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read certificate: %w", err)
	}

	key, err := readKey(pkiPath, name)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read key: %w", err)
	}

	return cert, key, nil
}

// readCert reads certificate from the specified location
func readCert(pkiPath, name string) (*x509.Certificate, error) {
	certificatePath := pathForCert(pkiPath, name)
	certBytes, err := os.ReadFile(certificatePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read certificate from file %s: %w", certificatePath, err)
	}
	return decodeCertPEM(certBytes)
}

// readKey reads key from the specified location
func readKey(pkiPath, name string) (crypto.Signer, error) {
	keyPath := pathForKey(pkiPath, name)
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read key from file %s: %w", keyPath, err)
	}
	return decodeKeyPEM(keyBytes)
}

// decodeCertPEM decodes a PEM-encoded certificate.
func decodeCertPEM(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

// decodeKeyPEM decodes a PEM-encoded key.
func decodeKeyPEM(keyPEM []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to decode key PEM")
	}
	switch block.Type {
	case RSAPrivateKeyBlockType:
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case ECPrivateKeyBlockType:
		return x509.ParseECPrivateKey(block.Bytes)
	}
	return nil, fmt.Errorf("unsupported key type %q", block.Type)
}

// WriteCertAndKey stores certificate and key at the specified location
func WriteCertAndKey(pkiPath string, name string, cert *x509.Certificate, key crypto.Signer) error {
	if err := writeKey(pkiPath, name, key); err != nil {
		return fmt.Errorf("couldn't write key: %w", err)
	}
	return writeCert(pkiPath, name, cert)
}

// writeCert stores the given certificate at the given location
func writeCert(pkiPath, name string, cert *x509.Certificate) error {
	certificatePath := pathForCert(pkiPath, name)
	encoded := EncodeCertToPEM(cert)
	if err := writeFile(certificatePath, encoded); err != nil {
		return fmt.Errorf("unable to write certificate to file %s: %w", certificatePath, err)
	}
	return nil
}

// writeKey stores the given key at the given location
func writeKey(pkiPath, name string, key crypto.Signer) error {
	privateKeyPath := pathForKey(pkiPath, name)
	encoded, err := EncodePrivateKeyToPEM(key)
	if err != nil {
		return fmt.Errorf("unable to marshal private key to PEM: %w", err)
	}
	if err := writeFile(privateKeyPath, encoded); err != nil {
		return fmt.Errorf("unable to write private key to file %s: %w", privateKeyPath, err)
	}
	return nil
}

// EncodeCertToPEM returns PEM-encoded certificate data
func EncodeCertToPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  CertificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// EncodePrivateKeyToPEM converts a known private key type of RSA or ECDSA to
// a PEM encoded block or returns an error.
func EncodePrivateKeyToPEM(privateKey crypto.PrivateKey) ([]byte, error) {
	switch t := privateKey.(type) {
	case *ecdsa.PrivateKey:
		derBytes, err := x509.MarshalECPrivateKey(t)
		if err != nil {
			return nil, err
		}
		block := &pem.Block{
			Type:  ECPrivateKeyBlockType,
			Bytes: derBytes,
		}
		return pem.EncodeToMemory(block), nil
	case *rsa.PrivateKey:
		block := &pem.Block{
			Type:  RSAPrivateKeyBlockType,
			Bytes: x509.MarshalPKCS1PrivateKey(t),
		}
		return pem.EncodeToMemory(block), nil
	default:
		return nil, fmt.Errorf("private key is not a recognized type: %T", privateKey)
	}
}

func writeFile(certPath string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(certPath), os.FileMode(0750)); err != nil {
		return err
	}
	return os.WriteFile(certPath, data, os.FileMode(0644))
}

func pathForKey(pkiPath, name string) string {
	return path.Join(pkiPath, fmt.Sprintf("%s.key", name))
}

func pathForCert(pkiPath, name string) string {
	return path.Join(pkiPath, fmt.Sprintf("%s.crt", name))
}

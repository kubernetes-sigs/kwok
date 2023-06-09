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
	"fmt"
	"testing"
	"time"
)

func TestGeneratePki(t *testing.T) {
	now := time.Now()
	notBefore := now.UTC()
	notAfter := now.Add(CertificateValidity).UTC()

	caCert, caKey, err := GenerateCA("kwok-ca", notBefore, notAfter)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to generate CA: %w", err))
	}

	cert, key, err := GenerateSignCert("kwok-admin", caCert, caKey, notBefore, notAfter, DefaultGroups, DefaultAltNames)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to generate admin cert and key: %w", err))
	}

	_, err = EncodePrivateKeyToPEM(key)
	if err != nil {
		t.Fatal(fmt.Errorf("failed to encode private key: %w", err))
	}

	_ = EncodeCertToPEM(cert)
}

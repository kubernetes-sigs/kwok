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

package testdata

// This is just a local key, it doesn't matter if it is leaked.

//go:generate openssl genrsa -out ca.key 2048
//go:generate openssl req -sha256 -x509 -new -nodes -key ca.key -subj "/CN=kwok-ca" -out ca.crt -days 365000
//go:generate openssl genrsa -out admin.key 2048
//go:generate openssl req -new -key admin.key -subj "/CN=kwok-admin/O=system:masters" -out admin.csr -config openssl.cnf
//go:generate openssl x509 -sha256 -req -in admin.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out admin.crt -days 365000 -extensions v3_req -extfile openssl.cnf

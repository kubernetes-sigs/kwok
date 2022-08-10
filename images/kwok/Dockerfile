# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM docker.io/library/golang:1.18-alpine AS builder
WORKDIR /go/src/sigs.k8s.io/kwok
COPY . .

# Check that if no vendor exists, install git in preparation for pulling the vendor later
RUN ls vendor || apk add -U git

ENV CGO_ENABLED=0
RUN go install ./cmd/kwok

FROM docker.io/library/alpine:3.16
COPY --from=builder /go/bin/kwok /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/kwok"]

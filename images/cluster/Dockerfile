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

FROM --platform=$TARGETPLATFORM docker.io/library/alpine:3.16
ARG TARGETPLATFORM
COPY --chmod=0755 bin/$TARGETPLATFORM/kwok /usr/local/bin/
COPY --chmod=0755 bin/$TARGETPLATFORM/kwokctl /usr/local/bin/
COPY --chmod=0755 images/cluster/entrypoint.sh /entrypoint.sh

ENV KWOK_CONTROLLER_BINARY /usr/local/bin/kwok
ENV KWOK_APISERVER_PORT 8080

ARG kube_version
ENV KWOK_KUBE_VERSION $kube_version

RUN echo "KUBE_VERSION: ${KWOK_KUBE_VERSION}" && \
    kwokctl create cluster --quiet-pull && \
    kwokctl kubectl version && \
    kwokctl delete cluster

ENTRYPOINT ["/entrypoint.sh"]

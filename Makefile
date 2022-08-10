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

GO_CMD ?= go

.PHONY: unit-test
unit-test:
	$(GO_CMD) test ./pkg/...

.PHONY: verify
verify:
	@echo "Verify go.mod & go.sum"
	$(GO_CMD) mod tidy
	git --no-pager diff --exit-code go.mod go.sum

	@echo "Verify gofmt"
	@out=`gofmt -l -d $$(find . -name '*.go')`; \
	if [ -n "$$out" ]; then \
		echo "$$out"; \
		exit 1; \
	fi

.PHONY: build
build:
	$(GO_CMD) build -o bin/kwok ./cmd/kwok/*.go

.PHONY: integration-test
integration-test:
	@echo "Not implemented yet"

.PHONY: e2e-test
e2e-test:
	./hack/e2e-test.sh

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

HUGO_VERSION = $(shell grep -E '^[[:space:]]*HUGO_VERSION' ../netlify.toml | cut -d '=' -f 2 | tr -d '[:space:]' | tr -d '"')
HUGO_IMAGE   = docker.io/floryn90/hugo:$(HUGO_VERSION)-ext
HUGO_BINARY  = $(shell which hugo)
HUGO_DOCKER  = docker run --rm -it --volume $(realpath $(CURDIR)/..):/src -p 1313:1313 --workdir /src/site --entrypoint=hugo $(HUGO_IMAGE)
HUGO		 = $(if $(HUGO_BINARY),$(HUGO_BINARY),$(HUGO_DOCKER))

.PHONY: serve
serve:
	$(HUGO) server --bind="0.0.0.0" \
	--ignoreCache \
	--buildFuture \
	--disableFastRender

.PHONY: build
build:
	$(HUGO) --enableGitInfo --buildFuture --minify

.PHONY: check
check: check-version

.PHONY: check-version
check-version: check-binary-version check-image-version

.PHONY: check-image-version
check-image-version:
	skopeo --override-os linux inspect docker://$(HUGO_IMAGE)

.PHONY: check-binary-version
check-binary-version:
	curl -sfL https://api.github.com/repos/gohugoio/hugo/releases/tags/v$(HUGO_VERSION)

.PHONY: links
links:
	@echo "Creating symlinks"
	@mkdir -p ./static/img/demo
	@for f in ../demo/*.svg; do \
		ln -s ../../../"$$f" ./static/img/demo/; \
	done
	@ ln -s ../../logo/kwok.png ./static/favicon.png
	@ ln -s ../../logo/kwok.svg ./static/favicon.svg

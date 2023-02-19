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

HUGO_VERSION = 0.110.0
HUGO_IMAGE = docker.io/klakegg/hugo:$(HUGO_VERSION)-ext
HUGO_BINARY  = $(shell which hugo)
HUGO_DOCKER  = docker run --rm -it --volume $(realpath $(CURDIR)/..):/src -p 1313:1313 --workdir /src/site --entrypoint=hugo $(HUGO_IMAGE)
HUGO		 = $(if $(HUGO_BINARY),$(HUGO_BINARY),$(HUGO_DOCKER))

.PHONY: serve
serve: themes/hugo-book
	$(HUGO) server --bind="0.0.0.0" \
	--ignoreCache \
	--buildFuture \
	--disableFastRender

.PHONY: build
build: themes/hugo-book
	$(HUGO) --enableGitInfo --buildFuture --minify

themes/hugo-book:
	git clone --depth 1 --branch v9 https://github.com/alex-shpak/hugo-book $@

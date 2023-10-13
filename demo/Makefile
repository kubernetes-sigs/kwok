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

SVG_TERM=./node_modules/.bin/svg-term
SVG_TO_VIDEO=./node_modules/.bin/svg-to-video

$(SVG_TERM):
	pip3 install asciinema
	npm i --save-dev svg-term-cli

$(SVG_TO_VIDEO):
	npm i --save-dev @wzshiming/svg-to-video

%.svg: %.demo
	@$(MAKE) $(SVG_TERM)
	@echo "Generating $@"
	@$(SVG_TERM) --out "$@" \
	  --window \
	  --command "cd .. && ./demo/play-demo.sh ./demo/$<"

%.mp4: %.svg
	@$(MAKE) $(SVG_TO_VIDEO)
	@echo "Generating $@"
	@$(SVG_TO_VIDEO) $< $@ \
	  --delay-start 1 \
	  --headless

.PHONY: all
all: $(patsubst %.demo,%.svg,$(wildcard *.demo))

.PHONY: clean
clean:
	@-rm -f *.svg

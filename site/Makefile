
DOCKER       = docker
HUGO_VERSION = 0.107.0
DOCKER_IMAGE = docker.io/klakegg/hugo:$(HUGO_VERSION)-ext
DOCKER_RUN   = $(DOCKER) run --rm -it --volume $(realpath $(CURDIR)/..):/src -p 1313:1313 --workdir /src/site --entrypoint=hugo $(DOCKER_IMAGE)

.PHONY: serve
serve: themes/hugo-book
	$(DOCKER_RUN) server --bind="0.0.0.0" \
	--ignoreCache \
	--buildFuture \
	--disableFastRender

.PHONY: build
build: themes/hugo-book
	$(DOCKER_RUN) --enableGitInfo --buildFuture --minify


themes/hugo-book:
	git clone --depth 1 --branch v9 https://github.com/alex-shpak/hugo-book themes/hugo-book

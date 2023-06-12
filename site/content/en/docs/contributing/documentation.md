---
title: Documentation
---

# Documentation

{{< hint "info" >}}

This document provides details on how to contribute to the documentation.

{{< /hint >}}

Our docs are built with [hugo] just like [kubernetes.io].
We provide a makefile for development that uses hugo.

## Directory Structure

- site
  - content - Markdown content source
    - en
      - docs
        - contributing - Contributing documentation
        - design - Design documentation
        - user - User documentation
      - menu - Left navigation menu
      - posts - Blog posts
  - static - Static assets

## Creating a new page

To create a new page, create a new markdown file under the appropriate directory, and add a link to it in the appropriate menu.

## Building

For simple content changes you can also just edit the markdown sources and send a pull request.
A build preview will be created and reply on your pull request.

For more involved documentation development, you can run `make -C site serve` from the [kwok repository] to run a local instance of the documentation, browsable at [http://localhost:1313](http://localhost:1313).

[hugo]: https://gohugo.io/
[kubernetes.io]: https://kubernetes.io/
[kwok repository]: https://github.com/kubernetes-sigs/kwok

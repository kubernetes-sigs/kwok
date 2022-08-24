package consts

import (
	"runtime"
)

var (
	Version      = "unknown"
	ImagePrefix  = "registry.k8s.io/kwok"
	BinaryPrefix = "https://github.com/kubernetes-sigs/kwok/releases/download"
	BinaryName   = "kwok-" + runtime.GOOS + "-" + runtime.GOARCH
)

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

package main

import (
	"context"
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	ctrcmd "sigs.k8s.io/kwok/pkg/kwok/cmd"
	ctlcmd "sigs.k8s.io/kwok/pkg/kwokctl/cmd"
)

func main() {
	// set HOME env var so that default values involve user's home directory do not depend on the running user.
	os.Setenv("HOME", "/home/user")
	os.Setenv("XDG_CONFIG_HOME", "/home/user/.config")

	err := doc.GenMarkdownTree(ctlcmd.NewCommand(context.TODO()), "./docs/userguide/kwokctl")
	if err != nil {
		log.Fatal(err)
	}

	err = doc.GenMarkdownTree(ctrcmd.NewCommand(context.TODO()), "./docs/userguide/kwok")
	if err != nil {
		log.Fatal(err)
	}
}

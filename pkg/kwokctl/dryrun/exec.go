/*
Copyright 2026 The Kubernetes Authors.

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

package dryrun

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

// PrintExec prints the command to be executed to the output stream.
func PrintExec(ctx context.Context, name string, args ...string) {
	_, _ = fmt.Fprintln(stdout, formatExec(ctx, name, args...))
}

func formatExec(ctx context.Context, name string, args ...string) string {
	const sep = " \\\n  "
	opt := utilsexec.GetExecOptions(ctx)
	out := bytes.NewBuffer(nil)
	if opt.Dir != "" {
		_, _ = fmt.Fprintf(out, "cd %s &&"+sep, opt.Dir)
	}

	for _, env := range opt.Env {
		_, _ = fmt.Fprintf(out, "%s"+sep, env)
	}

	_, _ = fmt.Fprintf(out, "%s"+sep, utilspath.OnlyName(name))

	for _, arg := range args {
		_, _ = fmt.Fprintf(out, "%s"+sep, arg)
	}

	outfile, ok := IsCatToFileWriter(opt.Out)
	if ok {
		_, _ = fmt.Fprintf(out, ">%s"+sep, outfile)
	}

	if erroutfile, ok := IsCatToFileWriter(opt.ErrOut); ok {
		if erroutfile == outfile {
			_, _ = fmt.Fprintf(out, "2>&1"+sep)
		} else {
			_, _ = fmt.Fprintf(out, "2>%s"+sep, outfile)
		}
	}

	if opt.Fork {
		_, _ = fmt.Fprintf(out, "&"+sep)
	}

	return strings.TrimSuffix(out.String(), sep)
}

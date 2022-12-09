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

package log

import (
	"context"
	"os"

	"github.com/spf13/pflag"
)

func InitFlags(ctx context.Context, flags *pflag.FlagSet) (context.Context, *Logger) {
	v := flags.IntP("v", "v", 0, "number for the log level verbosity")
	_ = flags.Parse(os.Args[1:])
	logger := NewLogger(os.Stdout, Level(*v))
	return NewContext(ctx, logger), logger
}

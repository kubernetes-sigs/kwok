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

package kind

import (
	"context"
	"fmt"
	"io"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	utilsexec "sigs.k8s.io/kwok/pkg/utils/exec"
	"sigs.k8s.io/kwok/pkg/utils/file"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

func (c *Cluster) logs(ctx context.Context, name string, out io.Writer, follow bool) error {
	componentName := c.getComponentName(name)

	args := []string{"logs", "-n", "kube-system"}
	if follow {
		args = append(args, "-f")
	}
	args = append(args, componentName)
	if c.IsDryRun() && !follow {
		if file, ok := dryrun.IsCatToFileWriter(out); ok {
			dryrun.PrintMessagef("%s >%s", runtime.FormatExec(ctx, "kubectl", args...), file)
			return nil
		}
	}

	err := c.Kubectl(utilsexec.WithAllWriteTo(ctx, out), args...)
	if err != nil {
		return err
	}
	return nil
}

// Logs returns the logs of the specified component.
func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, false)
}

// LogsFollow follows the logs of the component
func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	return c.logs(ctx, name, out, true)
}

// CollectLogs returns the logs of the specified component.
func (c *Cluster) CollectLogs(ctx context.Context, dir string) error {
	logger := log.FromContext(ctx)

	kwokConfigPath := utilspath.Join(dir, "kwok.yaml")
	if file.Exists(kwokConfigPath) {
		return fmt.Errorf("%s already exists", kwokConfigPath)
	}

	if err := c.MkdirAll(dir); err != nil {
		return fmt.Errorf("failed to create tmp directory: %w", err)
	}
	logger.Info("Exporting logs",
		"dir", dir,
	)

	err := c.CopyFile(c.GetWorkdirPath(runtime.ConfigName), kwokConfigPath)
	if err != nil {
		return err
	}

	conf, err := c.Config(ctx)
	if err != nil {
		return err
	}

	componentsDir := utilspath.Join(dir, "components")
	err = c.MkdirAll(componentsDir)
	if err != nil {
		return err
	}

	kindPath, err := c.preDownloadKind(ctx)
	if err != nil {
		return err
	}

	infoPath := utilspath.Join(dir, conf.Options.Runtime+"-info.txt")
	err = c.WriteToPath(c.withProviderEnv(ctx), infoPath, []string{kindPath, "version"})
	if err != nil {
		return err
	}

	for _, component := range conf.Components {
		logPath := utilspath.Join(componentsDir, component.Name+".log")
		f, err := c.OpenFile(logPath)
		if err != nil {
			logger.Error("Failed to open file",
				"err", err,
			)
			continue
		}
		if err = c.Logs(ctx, component.Name, f); err != nil {
			logger.Error("Failed to get log",
				"err", err,
			)
			if err = f.Close(); err != nil {
				logger.Error("Failed to close file",
					"err", err,
				)
				if err = c.Remove(logPath); err != nil {
					logger.Error("Failed to remove file",
						"err", err,
					)
				}
			}
		}
		if err = f.Close(); err != nil {
			logger.Error("Failed to close file",
				"err", err,
			)
			if err = c.Remove(logPath); err != nil {
				logger.Error("Failed to remove file",
					"err", err,
				)
			}
		}
	}

	if conf.Options.KubeAuditPolicy != "" {
		filePath := utilspath.Join(componentsDir, "audit.log")
		f, err := c.OpenFile(filePath)
		if err != nil {
			logger.Error("Failed to open file",
				"err", err,
			)
		} else {
			if err = c.AuditLogs(ctx, f); err != nil {
				logger.Error("Failed to get audit log",
					"err", err,
				)
			}
			if err = f.Close(); err != nil {
				logger.Error("Failed to close file",
					"err", err,
				)
				if err = c.Remove(filePath); err != nil {
					logger.Error("Failed to remove file",
						"err", err,
					)
				}
			}
		}
	}

	return nil
}

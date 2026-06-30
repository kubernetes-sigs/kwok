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

package binary

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/nxadm/tail"

	"sigs.k8s.io/kwok/pkg/config"
	"sigs.k8s.io/kwok/pkg/consts"
	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/kwokctl/runtime"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/file"
	utilspath "sigs.k8s.io/kwok/pkg/utils/path"
)

// Logs returns the logs of the specified component.
func (c *Cluster) Logs(ctx context.Context, name string, out io.Writer) error {
	_, err := c.GetComponent(ctx, name)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)

	logs := c.GetLogPath(name + ".log")
	if c.IsDryRun() {
		if file, ok := dryrun.IsCatToFileWriter(out); ok {
			dryrun.PrintMessagef("cp %s %s", logs, file)
		} else {
			dryrun.PrintMessagef("cat %s", logs)
		}
		return nil
	}

	f, err := os.OpenFile(logs, os.O_RDONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", logs, err)
	}
	defer func() {
		err = f.Close()
		if err != nil {
			logger.Error("Failed to close file",
				"err", err,
			)
		}
	}()

	_, err = io.Copy(out, f)
	if err != nil {
		return err
	}
	return nil
}

// LogsFollow follows the logs of the component
func (c *Cluster) LogsFollow(ctx context.Context, name string, out io.Writer) error {
	_, err := c.GetComponent(ctx, name)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)

	logs := c.GetLogPath(name + ".log")
	if c.IsDryRun() {
		dryrun.PrintMessagef("tail -f %s", logs)
		return nil
	}

	t, err := tail.TailFile(logs, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}
	defer func() {
		err = t.Stop()
		if err != nil {
			logger.Error("Failed to stop tail file",
				"err", err,
			)
		}
	}()

	go func() {
		for line := range t.Lines {
			_, err = out.Write([]byte(line.Text + "\n"))
			if err != nil {
				logger.Error("Failed to write line text",
					"err", err,
				)
			}
		}
	}()
	<-ctx.Done()
	return nil
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

	infoPath := utilspath.Join(dir, consts.RuntimeTypeBinary+"-info.txt")
	f, err := c.OpenFile(infoPath)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, "%s/%s", config.GOOS, config.GOARCH)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}

	for _, component := range conf.Components {
		src := c.GetLogPath(component.Name + ".log")
		dest := utilspath.Join(componentsDir, component.Name+".log")
		if err = c.CopyFile(src, dest); err != nil {
			logger.Error("Failed to copy file",
				"err", err,
			)
		}
	}
	if conf.Options.KubeAuditPolicy != "" {
		src := c.GetLogPath(runtime.AuditLogName)
		dest := utilspath.Join(componentsDir, runtime.AuditLogName)
		if err = c.CopyFile(src, dest); err != nil {
			logger.Error("Failed to copy file",
				"err", err,
			)
		}
	}

	return nil
}

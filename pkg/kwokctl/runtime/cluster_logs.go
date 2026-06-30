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

package runtime

import (
	"context"
	"io"
	"os"

	"github.com/nxadm/tail"

	"sigs.k8s.io/kwok/pkg/kwokctl/dryrun"
	"sigs.k8s.io/kwok/pkg/log"
)

// AuditLogs returns the audit logs of the cluster.
func (c *Cluster) AuditLogs(ctx context.Context, out io.Writer) error {
	logs := c.GetLogPath(AuditLogName)
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
		return err
	}
	logger := log.FromContext(ctx)
	defer func() {
		err = f.Close()
		if err != nil {
			logger.Error("Failed to close file",
				"err", err,
			)
		}
	}()

	_, err = io.Copy(out, f)
	return err
}

// AuditLogsFollow follows the audit logs of the cluster.
func (c *Cluster) AuditLogsFollow(ctx context.Context, out io.Writer) error {
	logs := c.GetLogPath(AuditLogName)
	if c.IsDryRun() {
		dryrun.PrintMessagef("tail -f %s", logs)
		return nil
	}

	t, err := tail.TailFile(logs, tail.Config{ReOpen: true, Follow: true})
	if err != nil {
		return err
	}
	logger := log.FromContext(ctx)
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

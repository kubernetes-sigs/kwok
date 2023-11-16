/*
Copyright 2023 The Kubernetes Authors.

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

package server

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/utils/format"
)

func Test_findLogInLogs(t *testing.T) {
	type args struct {
		containerName string
		logs          []internalversion.Log
	}
	tests := []struct {
		name   string
		args   args
		want   *internalversion.Log
		wantOk bool
	}{
		{
			name: "find log in logs",
			args: args{
				containerName: "test",
				logs: []internalversion.Log{
					{
						Containers: []string{"test"},
					},
				},
			},
			want: &internalversion.Log{
				Containers: []string{"test"},
			},
			wantOk: true,
		},
		{
			name: "not find log in logs",
			args: args{
				containerName: "test",
				logs: []internalversion.Log{
					{
						Containers: []string{"test1"},
					},
				},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "not find log in empty logs",
			args: args{
				containerName: "test",
				logs:          []internalversion.Log{},
			},
			want:   nil,
			wantOk: false,
		},
		{
			name: "find log in empty logs",
			args: args{
				containerName: "test",
				logs: []internalversion.Log{
					{
						Containers: []string{},
					},
				},
			},
			want: &internalversion.Log{
				Containers: []string{},
			},
			wantOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotOk := findLogInLogs(tt.args.containerName, tt.args.logs)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findLogInLogs() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("findLogInLogs() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func Test_getPodLogs(t *testing.T) {
	type args struct {
		rules         []*internalversion.Logs
		clusterRules  []*internalversion.ClusterLogs
		podName       string
		podNamespace  string
		containerName string
	}
	tests := []struct {
		name    string
		args    args
		want    *internalversion.Log
		wantErr bool
	}{
		{
			name: "find logs in rule",
			args: args{
				rules: []*internalversion.Logs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test",
							Namespace: "default",
						},
						Spec: internalversion.LogsSpec{
							Logs: []internalversion.Log{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterLogs{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want: &internalversion.Log{
				Containers: []string{"test"},
			},
		},
		{
			name: "find logs in cluster rule",
			args: args{
				rules: []*internalversion.Logs{},
				clusterRules: []*internalversion.ClusterLogs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterLogsSpec{
							Logs: []internalversion.Log{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want: &internalversion.Log{
				Containers: []string{"test"},
			},
		},
		{
			name: "not find logs in rule",
			args: args{
				rules: []*internalversion.Logs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-2",
							Namespace: "default",
						},
						Spec: internalversion.LogsSpec{
							Logs: []internalversion.Log{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				clusterRules:  []*internalversion.ClusterLogs{},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not find logs in cluster rule",
			args: args{
				rules: []*internalversion.Logs{},
				clusterRules: []*internalversion.ClusterLogs{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-test",
						},
						Spec: internalversion.ClusterLogsSpec{
							Selector: &internalversion.ObjectSelector{
								MatchNamespaces: []string{"test"},
							},
							Logs: []internalversion.Log{
								{
									Containers: []string{"test"},
								},
							},
						},
					},
				},
				podName:       "test",
				podNamespace:  "default",
				containerName: "test",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPodLogs(tt.args.rules, tt.args.clusterRules, tt.args.podName, tt.args.podNamespace, tt.args.containerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPodLogs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPodLogs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_logOptions(t *testing.T) {
	var (
		line         = int64(8)
		bytes        = int64(64)
		timestamp    = metav1.Now()
		sinceseconds = int64(10)
	)
	for c, test := range []struct {
		apiOpts *corev1.PodLogOptions
		expect  *logOptions
	}{
		{ // empty options
			apiOpts: &corev1.PodLogOptions{},
			expect:  &logOptions{tail: -1, bytes: -1},
		},
		{ // test tail lines
			apiOpts: &corev1.PodLogOptions{TailLines: &line},
			expect:  &logOptions{tail: line, bytes: -1},
		},
		{ // test limit bytes
			apiOpts: &corev1.PodLogOptions{LimitBytes: &bytes},
			expect:  &logOptions{tail: -1, bytes: bytes},
		},
		{ // test since timestamp
			apiOpts: &corev1.PodLogOptions{SinceTime: &timestamp},
			expect:  &logOptions{tail: -1, bytes: -1, since: timestamp.Time},
		},
		{ // test since seconds
			apiOpts: &corev1.PodLogOptions{SinceSeconds: &sinceseconds},
			expect:  &logOptions{tail: -1, bytes: -1, since: timestamp.Add(-10 * time.Second)},
		},
	} {
		t.Logf("TestCase #%d: %+v", c, test)
		opts := newLogOptions(test.apiOpts, timestamp.Time)
		if !reflect.DeepEqual(opts, test.expect) {
			t.Errorf("expected %+v, got %+v", test.expect, opts)
		}
	}
}

func Test_readLogs(t *testing.T) {
	file, err := os.CreateTemp("", "Test_readLogs")
	if err != nil {
		t.Fatalf("unable to create temp file")
	}
	defer func() {
		_ = os.Remove(file.Name())
	}()
	_, _ = file.WriteString(`{"log":"line1\n","stream":"stdout","time":"2020-09-27T11:18:01.00000000Z"}` + "\n")
	_, _ = file.WriteString(`{"log":"line2\n","stream":"stdout","time":"2020-09-27T11:18:02.00000000Z"}` + "\n")
	_, _ = file.WriteString(`{"log":"line3\n","stream":"stdout","time":"2020-09-27T11:18:03.00000000Z"}` + "\n")

	testCases := []struct {
		name          string
		podLogOptions corev1.PodLogOptions
		expected      string
	}{
		{
			name:          "default pod log options should output all lines",
			podLogOptions: corev1.PodLogOptions{},
			expected:      "line1\nline2\nline3\n",
		},
		{
			name: "using TailLines 2 should output last 2 lines",
			podLogOptions: corev1.PodLogOptions{
				TailLines: format.Ptr[int64](2),
			},
			expected: "line2\nline3\n",
		},
		{
			name: "using TailLines 4 should output all lines when the log has less than 4 lines",
			podLogOptions: corev1.PodLogOptions{
				TailLines: format.Ptr[int64](4),
			},
			expected: "line1\nline2\nline3\n",
		},
		{
			name: "using TailLines 0 should output nothing",
			podLogOptions: corev1.PodLogOptions{
				TailLines: format.Ptr[int64](0),
			},
			expected: "",
		},
		{
			name: "using LimitBytes 9 should output first 9 bytes",
			podLogOptions: corev1.PodLogOptions{
				LimitBytes: format.Ptr[int64](9),
			},
			expected: "line1\nlin",
		},
		{
			name: "using LimitBytes 100 should output all bytes when the log has less than 100 bytes",
			podLogOptions: corev1.PodLogOptions{
				LimitBytes: format.Ptr[int64](100),
			},
			expected: "line1\nline2\nline3\n",
		},
		{
			name: "using LimitBytes 0 should output nothing",
			podLogOptions: corev1.PodLogOptions{
				LimitBytes: format.Ptr[int64](0),
			},
			expected: "",
		},
		{
			name: "using SinceTime should output lines with a time on or after the specified time",
			podLogOptions: corev1.PodLogOptions{
				SinceTime: &metav1.Time{Time: time.Date(2020, time.Month(9), 27, 11, 18, 02, 0, time.UTC)},
			},
			expected: "line2\nline3\n",
		},
		{
			name: "using SinceTime now should output nothing",
			podLogOptions: corev1.PodLogOptions{
				SinceTime: &metav1.Time{Time: time.Now()},
			},
			expected: "",
		},
		{
			name: "using follow should output all log lines",
			podLogOptions: corev1.PodLogOptions{
				Follow: true,
			},
			expected: "line1\nline2\nline3\n",
		},
		{
			name: "using follow combined with TailLines 2 should output the last 2 lines",
			podLogOptions: corev1.PodLogOptions{
				Follow:    true,
				TailLines: format.Ptr[int64](2),
			},
			expected: "line2\nline3\n",
		},
		{
			name: "using follow combined with SinceTime should output lines with a time on or after the specified time",
			podLogOptions: corev1.PodLogOptions{
				Follow:    true,
				SinceTime: &metav1.Time{Time: time.Date(2020, time.Month(9), 27, 11, 18, 02, 0, time.UTC)},
			},
			expected: "line2\nline3\n",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			apiOpt := tc.podLogOptions
			opts := newLogOptions(&apiOpt, time.Now())
			stdoutBuf := bytes.NewBuffer(nil)
			stderrBuf := bytes.NewBuffer(nil)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			err = readLogs(ctx, file.Name(), opts, stdoutBuf, stderrBuf)

			if err != nil && !errors.Is(err, errContextCanceled) {
				t.Fatalf(err.Error())
			}
			if stderrBuf.Len() > 0 {
				t.Fatalf("Stderr: %v", stderrBuf.String())
			}
			if actual := stdoutBuf.String(); tc.expected != actual {
				t.Fatalf("Actual output does not match expected.\nActual:  %v\nExpected: %v\n", actual, tc.expected)
			}
		})
	}
}

func TestReadRotatedLog(t *testing.T) {
	tmpDir := t.TempDir()
	file, err := os.CreateTemp(tmpDir, "logfile")
	if err != nil {
		t.Fatalf("unable to create temp file")
	}

	stdoutBuf := &bytes.Buffer{}
	stderrBuf := &bytes.Buffer{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Start to follow the container's log.
	go func(ctx context.Context) {
		podLogOptions := corev1.PodLogOptions{
			Follow: true,
		}
		opts := newLogOptions(&podLogOptions, time.Now())
		_ = readLogs(ctx, file.Name(), opts, stdoutBuf, stderrBuf)
	}(ctx)

	// log in stdout
	expectedStdout := "line0\nline2\nline4\nline6\nline8\n"
	// log in stderr
	expectedStderr := "line1\nline3\nline5\nline7\nline9\n"

	dir := filepath.Dir(file.Name())
	baseName := filepath.Base(file.Name())

	// Write 10 lines to log file.
	// Let ReadLogs start.
	time.Sleep(50 * time.Millisecond)
	for line := 0; line < 10; line++ {
		// Write the first three lines to log file
		now := time.Now().Format(timeFormatIn)
		if line%2 == 0 {
			_, _ = file.WriteString(fmt.Sprintf(
				`{"log":"line%d\n","stream":"stdout","time":"%s"}`+"\n", line, now))
		} else {
			_, _ = file.WriteString(fmt.Sprintf(
				`{"log":"line%d\n","stream":"stderr","time":"%s"}`+"\n", line, now))
		}
		time.Sleep(1 * time.Millisecond)

		if line == 5 {
			_ = file.Close()
			// Pretend to rotate the log.
			rotatedName := fmt.Sprintf("%s.%s", baseName, time.Now().Format("220060102-150405"))
			rotatedName = filepath.Join(dir, rotatedName)
			if err := os.Rename(filepath.Join(dir, baseName), rotatedName); err != nil {
				t.Fatalf("failed to rotate log %q to %q: %v", file.Name(), rotatedName, err)
				return
			}

			newF := filepath.Join(dir, baseName)
			if file, err = os.Create(newF); err != nil {
				t.Fatalf("unable to create new log file: %v", err)
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	time.Sleep(20 * time.Millisecond)
	// Make the function ReadLogs end.

	if expectedStdout != stdoutBuf.String() {
		t.Fatalf("Stdout: %v", stdoutBuf.String())
	}
	if expectedStderr != stderrBuf.String() {
		t.Fatalf("Stderr: %v", stderrBuf.String())
	}
}

func TestParseLog(t *testing.T) {
	timestamp, err := time.Parse(timeFormatIn, "2016-10-20T18:39:20.57606443Z")
	if err != nil {
		t.Fatalf("unable to parse timestamp")
	}
	msg := &logMessage{}
	for c, test := range []struct {
		line string
		msg  *logMessage
		err  bool
	}{
		{ // Docker log format stdout
			line: `{"log":"docker stdout test log","stream":"stdout","time":"2016-10-20T18:39:20.57606443Z"}` + "\n",
			msg: &logMessage{
				timestamp: timestamp,
				stream:    runtimeapi.Stdout,
				log:       []byte("docker stdout test log"),
			},
		},
		{ // Docker log format stderr
			line: `{"log":"docker stderr test log","stream":"stderr","time":"2016-10-20T18:39:20.57606443Z"}` + "\n",
			msg: &logMessage{
				timestamp: timestamp,
				stream:    runtimeapi.Stderr,
				log:       []byte("docker stderr test log"),
			},
		},
		{ // CRI log format stdout
			line: "2016-10-20T18:39:20.57606443Z stdout F cri stdout test log\n",
			msg: &logMessage{
				timestamp: timestamp,
				stream:    runtimeapi.Stdout,
				log:       []byte("cri stdout test log\n"),
			},
		},
		{ // CRI log format stderr
			line: "2016-10-20T18:39:20.57606443Z stderr F cri stderr test log\n",
			msg: &logMessage{
				timestamp: timestamp,
				stream:    runtimeapi.Stderr,
				log:       []byte("cri stderr test log\n"),
			},
		},
		{ // Unsupported Log format
			line: "unsupported log format test log\n",
			msg:  &logMessage{},
			err:  true,
		},
		{ // Partial CRI log line
			line: "2016-10-20T18:39:20.57606443Z stdout P cri stdout partial test log\n",
			msg: &logMessage{
				timestamp: timestamp,
				stream:    runtimeapi.Stdout,
				log:       []byte("cri stdout partial test log"),
			},
		},
		{ // Partial CRI log line with multiple log tags.
			line: "2016-10-20T18:39:20.57606443Z stdout P:TAG1:TAG2 cri stdout partial test log\n",
			msg: &logMessage{
				timestamp: timestamp,
				stream:    runtimeapi.Stdout,
				log:       []byte("cri stdout partial test log"),
			},
		},
	} {
		t.Logf("TestCase #%d: %+v", c, test)
		parse, err := getParseFunc([]byte(test.line))
		if test.err {
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			continue
		}
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		err = parse([]byte(test.line), msg)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if !reflect.DeepEqual(msg, test.msg) {
			t.Errorf("expected %+v, got %+v", test.msg, msg)
		}
	}
}

func TestWriteLogs(t *testing.T) {
	timestamp := time.Unix(1234, 43210)
	log := "abcdefg\n"

	for c, test := range []struct {
		stream       runtimeapi.LogStreamType
		since        time.Time
		timestamp    bool
		expectStdout string
		expectStderr string
	}{
		{ // stderr log
			stream:       runtimeapi.Stderr,
			expectStderr: log,
		},
		{ // stdout log
			stream:       runtimeapi.Stdout,
			expectStdout: log,
		},
		{ // since is after timestamp
			stream: runtimeapi.Stdout,
			since:  timestamp.Add(1 * time.Second),
		},
		{ // timestamp enabled
			stream:       runtimeapi.Stderr,
			timestamp:    true,
			expectStderr: timestamp.Format(timeFormatOut) + " " + log,
		},
	} {
		t.Logf("TestCase #%d: %+v", c, test)
		msg := &logMessage{
			timestamp: timestamp,
			stream:    test.stream,
			log:       []byte(log),
		}
		stdoutBuf := bytes.NewBuffer(nil)
		stderrBuf := bytes.NewBuffer(nil)
		w := newLogWriter(stdoutBuf, stderrBuf, &logOptions{since: test.since, timestamp: test.timestamp, bytes: -1})
		err := w.write(msg, true)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if test.expectStdout != stdoutBuf.String() {
			t.Errorf("expected %q, got %q", test.expectStdout, stdoutBuf.String())
		}
		if test.expectStderr != stderrBuf.String() {
			t.Errorf("expected %q, got %q", test.expectStderr, stderrBuf.String())
		}
	}
}

func TestWriteLogsWithBytesLimit(t *testing.T) {
	timestamp := time.Unix(1234, 4321)
	timestampStr := timestamp.Format(timeFormatOut)
	log := "abcdefg\n"

	for c, test := range []struct {
		stdoutLines  int
		stderrLines  int
		bytes        int
		timestamp    bool
		expectStdout string
		expectStderr string
	}{
		{ // limit bytes less than one line
			stdoutLines:  3,
			bytes:        3,
			expectStdout: "abc",
		},
		{ // limit bytes across lines
			stdoutLines:  3,
			bytes:        len(log) + 3,
			expectStdout: "abcdefg\nabc",
		},
		{ // limit bytes more than all lines
			stdoutLines:  3,
			bytes:        3 * len(log),
			expectStdout: "abcdefg\nabcdefg\nabcdefg\n",
		},
		{ // limit bytes for stderr
			stderrLines:  3,
			bytes:        len(log) + 3,
			expectStderr: "abcdefg\nabc",
		},
		{ // limit bytes for both stdout and stderr, stdout first.
			stdoutLines:  1,
			stderrLines:  2,
			bytes:        len(log) + 3,
			expectStdout: "abcdefg\n",
			expectStderr: "abc",
		},
		{ // limit bytes with timestamp
			stdoutLines:  3,
			timestamp:    true,
			bytes:        len(timestampStr) + 1 + len(log) + 2,
			expectStdout: timestampStr + " " + log + timestampStr[:2],
		},
	} {
		t.Logf("TestCase #%d: %+v", c, test)
		msg := &logMessage{
			timestamp: timestamp,
			log:       []byte(log),
		}
		stdoutBuf := bytes.NewBuffer(nil)
		stderrBuf := bytes.NewBuffer(nil)
		w := newLogWriter(stdoutBuf, stderrBuf, &logOptions{timestamp: test.timestamp, bytes: int64(test.bytes)})
		for i := 0; i < test.stdoutLines; i++ {
			msg.stream = runtimeapi.Stdout
			if err := w.write(msg, true); err != nil {
				if !errors.Is(err, errMaximumWrite) {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}
		for i := 0; i < test.stderrLines; i++ {
			msg.stream = runtimeapi.Stderr
			if err := w.write(msg, true); err != nil {
				if !errors.Is(err, errMaximumWrite) {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}
		if test.expectStdout != stdoutBuf.String() {
			t.Errorf("expected %q, got %q", test.expectStdout, stdoutBuf.String())
		}
		if test.expectStderr != stderrBuf.String() {
			t.Errorf("expected %q, got %q", test.expectStderr, stderrBuf.String())
		}
	}
}

func TestReadLogsLimitsWithTimestamps(t *testing.T) {
	logLineFmt := "2022-10-29T16:10:22.592603036-05:00 stdout P %v\n"
	logLineNewLine := "2022-10-29T16:10:22.592603036-05:00 stdout F \n"

	tmpfile, err := os.CreateTemp("", "log.*.txt")
	if err != nil {
		t.Fatalf("unable to create temp file")
	}

	count := 10000

	for i := 0; i < count; i++ {
		_, _ = tmpfile.WriteString(fmt.Sprintf(logLineFmt, i))
	}
	_, _ = tmpfile.WriteString(logLineNewLine)

	for i := 0; i < count; i++ {
		_, _ = tmpfile.WriteString(fmt.Sprintf(logLineFmt, i))
	}
	_, _ = tmpfile.WriteString(logLineNewLine)

	// two lines are in the buffer

	defer func() {
		_ = os.Remove(tmpfile.Name()) // clean up
	}()

	_ = tmpfile.Close()

	var buf bytes.Buffer
	w := io.MultiWriter(&buf)

	err = readLogs(context.Background(), tmpfile.Name(), &logOptions{tail: -1, bytes: -1, timestamp: true}, w, w)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lineCount := 0
	scanner := bufio.NewScanner(bytes.NewReader(buf.Bytes()))
	for scanner.Scan() {
		lineCount++

		// Split the line
		ts, logline, _ := bytes.Cut(scanner.Bytes(), []byte(" "))

		// Verification
		//   1. The timestamp should exist
		//   2. The last item in the log should be 9999
		_, err = time.Parse(time.RFC3339, string(ts))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !bytes.HasSuffix(logline, []byte("9999")) {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

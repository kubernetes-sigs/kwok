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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/fsnotify/fsnotify"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiserver/pkg/util/flushwriter"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"

	"sigs.k8s.io/kwok/pkg/apis/internalversion"
	"sigs.k8s.io/kwok/pkg/log"
	"sigs.k8s.io/kwok/pkg/utils/slices"
	"sigs.k8s.io/kwok/pkg/utils/tail"
)

const (
	// timeFormatOut is the format for writing timestamps to output.
	// It is the fixed width version of time.RFC3339Nano.
	timeFormatOut = "2006-01-02T15:04:05.000000000Z07:00"

	// timeFormatIn is the format for parsing timestamps from other logs.
	// It is the variable width RFC3339 time format for lenient parsing of strings into timestamps.
	timeFormatIn = "2006-01-02T15:04:05.999999999Z07:00"

	// logForceCheckPeriod is the period to check for a new read
	logForceCheckPeriod = 1 * time.Second
)

var (
	// delimiter is the delimiter for timestamp and stream type in log line.
	delimiter = []byte{' '}
	// tagDelimiter is the delimiter for log tags.
	tagDelimiter = []byte(":")
)

// GetContainerLogs returns logs for a container in a pod.
// If follow is true, it streams the logs until the connection is closed by the client.
func (s *Server) GetContainerLogs(ctx context.Context, podName, podNamespace, container string, logOptions *corev1.PodLogOptions, stdout, stderr io.Writer) error {
	log, err := getPodLogs(s.logs.Get(), s.clusterLogs.Get(), podName, podNamespace, container)
	if err != nil {
		return err
	}

	opts := newLogOptions(logOptions, time.Now())
	return readLogs(ctx, log.LogsFile, opts, stdout, stderr)
}

// getContainerLogs handles containerLogs request against the Kubelet
func (s *Server) getContainerLogs(request *restful.Request, response *restful.Response) {
	podNamespace := request.PathParameter("podNamespace")
	podName := request.PathParameter("podID")
	containerName := request.PathParameter("containerName")

	if len(podName) == 0 {
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing podID."}`))
		return
	}
	if len(containerName) == 0 {
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing restfulCont name."}`))
		return
	}
	if len(podNamespace) == 0 {
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Missing podNamespace."}`))
		return
	}

	query := request.Request.URL.Query()
	// backwards compatibility for the "tail" query parameter
	if tail := request.QueryParameter("tail"); len(tail) > 0 {
		query["tailLines"] = []string{tail}
		// "all" is the same as omitting tail
		if tail == "all" {
			delete(query, "tailLines")
		}
	}

	// restfulCont logs on the kubelet are locked to the corev1 API version of PodLogOptions
	logOptions := &corev1.PodLogOptions{}
	err := convert_url_Values_To_v1_PodLogOptions(&query, logOptions, nil)
	if err != nil {
		logger := log.FromContext(request.Request.Context())
		logger.Error("Unable to decode the request for container logs", err)
		_ = response.WriteError(http.StatusBadRequest, fmt.Errorf(`{"message": "Unable to decode query."}`))
		return
	}

	if _, ok := response.ResponseWriter.(http.Flusher); !ok {
		_ = response.WriteError(http.StatusInternalServerError, fmt.Errorf("unable to convert %v into http.Flusher, cannot show logs", reflect.TypeOf(response)))
		return
	}
	fw := flushwriter.Wrap(response.ResponseWriter)
	response.Header().Set("Transfer-Encoding", "chunked")
	if err := s.GetContainerLogs(request.Request.Context(), podName, podNamespace, containerName, logOptions, fw, fw); err != nil {
		_ = response.WriteError(http.StatusBadRequest, err)
		return
	}
}

func getPodLogs(rules []*internalversion.Logs, clusterRules []*internalversion.ClusterLogs, podName, podNamespace, containerName string) (*internalversion.Log, error) {
	l, has := slices.Find(rules, func(l *internalversion.Logs) bool {
		return l.Name == podName && l.Namespace == podNamespace
	})
	if has {
		l, found := findLogInLogs(containerName, l.Spec.Logs)
		if found {
			return l, nil
		}
		return nil, fmt.Errorf("log target not found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
	}

	for _, cl := range clusterRules {
		if !cl.Spec.Selector.Match(podName, podNamespace) {
			continue
		}

		log, found := findLogInLogs(containerName, cl.Spec.Logs)
		if found {
			return log, nil
		}
	}
	return nil, fmt.Errorf("no logs found for container %q in pod %q", containerName, log.KRef(podNamespace, podName))
}

func findLogInLogs(containerName string, logs []internalversion.Log) (*internalversion.Log, bool) {
	var defaultLog *internalversion.Log
	for i, l := range logs {
		if len(l.Containers) == 0 && defaultLog == nil {
			defaultLog = &logs[i]
			continue
		}
		if slices.Contains(l.Containers, containerName) {
			return &l, true
		}
	}
	return defaultLog, defaultLog != nil
}

// logMessage is the CRI internal log type.
type logMessage struct {
	timestamp time.Time
	stream    runtimeapi.LogStreamType
	log       []byte
}

// reset resets the log to nil.
func (l *logMessage) reset() {
	l.timestamp = time.Time{}
	l.stream = ""
	l.log = nil
}

// logOptions is the CRI internal type of all log options.
type logOptions struct {
	tail      int64
	bytes     int64
	since     time.Time
	follow    bool
	timestamp bool
}

// newLogOptions convert the v1.PodLogOptions to CRI internal logOptions.
func newLogOptions(apiOpts *corev1.PodLogOptions, now time.Time) *logOptions {
	opts := &logOptions{
		tail:      -1, // -1 by default which means read all logs.
		bytes:     -1, // -1 by default which means read all logs.
		follow:    apiOpts.Follow,
		timestamp: apiOpts.Timestamps,
	}
	if apiOpts.TailLines != nil {
		opts.tail = *apiOpts.TailLines
	}
	if apiOpts.LimitBytes != nil {
		opts.bytes = *apiOpts.LimitBytes
	}
	if apiOpts.SinceSeconds != nil {
		opts.since = now.Add(-time.Duration(*apiOpts.SinceSeconds) * time.Second)
	}
	if apiOpts.SinceTime != nil && apiOpts.SinceTime.After(opts.since) {
		opts.since = apiOpts.SinceTime.Time
	}
	return opts
}

func readLogs(ctx context.Context, logsFile string, opts *logOptions, stdout, stderr io.Writer) error {
	logger := log.FromContext(ctx)

	f, err := os.Open(logsFile)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %w", logsFile, err)
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	// Search start point based on tail line.
	start, err := tail.FindTailLineStartIndex(f, opts.tail)
	if err != nil {
		return fmt.Errorf("failed to tail %d lines of log file %q: %w", opts.tail, logsFile, err)
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek %d in log file %q: %w", start, logsFile, err)
	}

	limitedMode := (opts.tail >= 0) && (!opts.follow)
	limitedNum := opts.tail
	// Start parsing the logs.
	r := bufio.NewReader(f)
	// Do not create watcher here because it is not needed if `Follow` is false.
	var watcher *fsnotify.Watcher
	var parse parseFunc
	var stop bool
	isNewLine := true
	found := true

	writer := newLogWriter(stdout, stderr, opts)
	msg := &logMessage{}
	baseName := filepath.Base(logsFile)
	dir := filepath.Dir(logsFile)
	for {
		if stop || (limitedMode && limitedNum == 0) {
			logger.Debug("Finished parsing log file", "path", logsFile)
			return nil
		}

		l, err := r.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) { // This is an real error
				return fmt.Errorf("failed to read log file %q: %w", logsFile, err)
			}

			if opts.follow {
				// The container is not running, we got to the end of the log.
				if !found {
					return nil
				}
				// Reset seek so that if this is an incomplete line,
				// it will be read again.
				if _, err := f.Seek(-int64(len(l)), io.SeekCurrent); err != nil {
					return fmt.Errorf("failed to reset seek in log file %q: %w", logsFile, err)
				}
				if watcher == nil {
					// Initialize the watcher if it has not been initialized yet.
					if watcher, err = fsnotify.NewWatcher(); err != nil {
						return fmt.Errorf("failed to create fsnotify watcher: %w", err)
					}
					defer func(watcher *fsnotify.Watcher) {
						_ = watcher.Close()
					}(watcher)
					if err := watcher.Add(dir); err != nil {
						return fmt.Errorf("failed to watch directory %q: %w", dir, err)
					}
					// If we just created the watcher, try again to read as we might have missed
					// the event.
					continue
				}
				var recreated bool
				// Wait until the next log change.
				found, recreated, err = waitLogs(ctx, baseName, watcher)
				if err != nil {
					return err
				}

				if recreated {
					newF, err := os.Open(logsFile)
					if err != nil {
						if os.IsNotExist(err) {
							continue
						}
						return fmt.Errorf("failed to open log file %q: %w", logsFile, err)
					}

					defer func(f *os.File) {
						_ = f.Close()
					}(newF)

					_ = f.Close()
					f = newF
					r = bufio.NewReader(f)
				}
				// If the container exited consume data until the next EOF
				continue
			}
			// Should stop after writing the remaining content.
			stop = true
			if len(l) == 0 {
				continue
			}
			logger.Info("Incomplete line in log file", "path", logsFile, "line", l)
		}

		if parse == nil {
			// Initialize the log parsing function.
			parse, err = getParseFunc(l)
			if err != nil {
				return fmt.Errorf("failed to get parse function: %w", err)
			}
		}
		// Parse the log line.
		msg.reset()
		if err := parse(l, msg); err != nil {
			logger.Error("Failed when parsing line in log file", err, "path", logsFile, "line", l)
			continue
		}

		// Write the log line into the stream.
		if err := writer.write(msg, isNewLine); err != nil {
			if errors.Is(err, errMaximumWrite) {
				logger.Debug("Finished parsing log file, hit bytes limit", "path", waitLogs, "limit", opts.bytes)
				return nil
			}
			logger.Error("Failed when writing line to log file", err, "path", logsFile, "line", msg)
			return err
		}

		if limitedMode {
			limitedNum--
		}

		if len(msg.log) > 0 {
			isNewLine = msg.log[len(msg.log)-1] == '\n'
		} else {
			isNewLine = true
		}
	}
}

// parseFunc is a function parsing one log line to the internal log type.
// Notice that the caller must make sure logMessage is not nil.
type parseFunc func([]byte, *logMessage) error

var parseFuncs = []parseFunc{
	parseCRILog,        // CRI log format parse function
	parseDockerJSONLog, // Docker JSON log format parse function
}

func getParseFunc(log []byte) (parseFunc, error) {
	for _, p := range parseFuncs {
		if err := p(log, &logMessage{}); err == nil {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unsupported log format: %q", log)
}

// parseCRILog parses logs in CRI log format. CRI Log format example:
//
//	2016-10-06T00:17:09.669794202Z stdout P log content 1
//	2016-10-06T00:17:09.669794203Z stderr F log content 2
func parseCRILog(log []byte, msg *logMessage) error {
	var err error
	// Parse timestamp
	idx := bytes.Index(log, delimiter)
	if idx < 0 {
		return fmt.Errorf("timestamp is not found")
	}
	msg.timestamp, err = time.Parse(timeFormatIn, string(log[:idx]))
	if err != nil {
		return fmt.Errorf("unexpected timestamp format %q: %w", timeFormatIn, err)
	}

	// Parse stream type
	log = log[idx+1:]
	idx = bytes.Index(log, delimiter)
	if idx < 0 {
		return fmt.Errorf("stream type is not found")
	}
	msg.stream = runtimeapi.LogStreamType(log[:idx])
	if msg.stream != runtimeapi.Stdout && msg.stream != runtimeapi.Stderr {
		return fmt.Errorf("unexpected stream type %q", msg.stream)
	}

	// Parse log tag
	log = log[idx+1:]
	idx = bytes.Index(log, delimiter)
	if idx < 0 {
		return fmt.Errorf("log tag is not found")
	}
	// Keep this forward compatible.
	tags := bytes.Split(log[:idx], tagDelimiter)
	partial := runtimeapi.LogTag(tags[0]) == runtimeapi.LogTagPartial
	// Trim the tailing new line if this is a partial line.
	if partial && len(log) > 0 && log[len(log)-1] == '\n' {
		log = log[:len(log)-1]
	}

	// Get log content
	msg.log = log[idx+1:]

	return nil
}

// jsonLog is a log message, typically a single entry from a given log stream.
// since the data structure is originally from docker, we should be careful to
// with any changes to jsonLog
type jsonLog struct {
	// Log is the log message
	Log string `json:"log,omitempty"`
	// Stream is the log source
	Stream string `json:"stream,omitempty"`
	// Created is the created timestamp of log
	Created time.Time `json:"time"`
}

// parseDockerJSONLog parses logs in Docker JSON log format. Docker JSON log format
// example:
//
//	{"log":"content 1","stream":"stdout","time":"2016-10-20T18:39:20.57606443Z"}
//	{"log":"content 2","stream":"stderr","time":"2016-10-20T18:39:20.57606444Z"}
func parseDockerJSONLog(log []byte, msg *logMessage) error {
	l := &jsonLog{}

	if err := json.Unmarshal(log, l); err != nil {
		return fmt.Errorf("failed with %w to unmarshal log %q", err, l)
	}
	msg.timestamp = l.Created
	msg.stream = runtimeapi.LogStreamType(l.Stream)
	msg.log = []byte(l.Log)
	return nil
}

// logWriter controls the writing into the stream based on the log options.
type logWriter struct {
	stdout io.Writer
	stderr io.Writer
	opts   *logOptions
	remain int64
}

// writeLogs writes logs into stdout, stderr.
func (w *logWriter) write(msg *logMessage, addPrefix bool) error {
	if msg.timestamp.Before(w.opts.since) {
		// Skip the line because it's older than since
		return nil
	}
	line := msg.log
	if w.opts.timestamp && addPrefix {
		prefix := append([]byte(msg.timestamp.Format(timeFormatOut)), delimiter[0])
		line = append(prefix, line...)
	}
	// If the line is longer than the remaining bytes, cut it.
	if int64(len(line)) > w.remain {
		line = line[:w.remain]
	}
	// Get the proper stream to write to.
	var stream io.Writer
	switch msg.stream {
	case runtimeapi.Stdout:
		stream = w.stdout
	case runtimeapi.Stderr:
		stream = w.stderr
	default:
		return fmt.Errorf("unexpected stream type %q", msg.stream)
	}
	n, err := stream.Write(line)
	w.remain -= int64(n)
	if err != nil {
		return err
	}
	// If the line has not been fully written, return errShortWrite
	if n < len(line) {
		return errShortWrite
	}
	// If there are no more bytes left, return errMaximumWrite
	if w.remain <= 0 {
		return errMaximumWrite
	}
	return nil
}

// errMaximumWrite is returned when all bytes have been written.
var errMaximumWrite = errors.New("maximum write")

// errShortWrite is returned when the message is not fully written.
var errShortWrite = errors.New("short write")

func newLogWriter(stdout io.Writer, stderr io.Writer, opts *logOptions) *logWriter {
	w := &logWriter{
		stdout: stdout,
		stderr: stderr,
		opts:   opts,
		remain: math.MaxInt64, // initialize it as infinity
	}
	if opts.bytes >= 0 {
		w.remain = opts.bytes
	}
	return w
}

var errContextCanceled = fmt.Errorf("context canceled")

// waitLogs wait for the next log write. It returns two booleans and an error. The first boolean
// indicates whether a new log is found; the second boolean if the log file was recreated;
// the error is error happens during waiting new logs.
func waitLogs(ctx context.Context, logName string, watcher *fsnotify.Watcher) (bool, bool, error) {
	logger := log.FromContext(ctx)
	errRetry := 5
	for {
		select {
		case <-ctx.Done():
			return false, false, errContextCanceled
		case e := <-watcher.Events:
			switch e.Op {
			case fsnotify.Write, fsnotify.Rename, fsnotify.Remove, fsnotify.Chmod:
				return true, false, nil
			case fsnotify.Create:
				return true, filepath.Base(e.Name) == logName, nil
			default:
				logger.Error("Received unexpected fsnotify event, retrying", nil, "event", e)
			}
		case err := <-watcher.Errors:
			logger.Error("Received fsnotify watch error, retrying unless no more retries left", err, "retries", errRetry)
			if errRetry == 0 {
				return false, false, err
			}
			errRetry--
		case <-time.After(logForceCheckPeriod):
			return true, false, nil
		}
	}
}

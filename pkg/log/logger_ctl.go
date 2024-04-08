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
	"encoding/json"
	"fmt"
	"io"
	"log/slog" //nolint:depguard
	"strconv"
	"strings"
	"unicode"

	"github.com/wzshiming/ctc"
	"golang.org/x/term"

	"sigs.k8s.io/kwok/pkg/utils/format"
	"sigs.k8s.io/kwok/pkg/utils/monospace"
)

type ctlHandler struct {
	level    Level
	output   io.Writer
	attrs    []slog.Attr
	attrsStr *string
	groups   []string
	fd       int
}

func newCtlHandler(w io.Writer, fd int, level Level) *ctlHandler {
	return &ctlHandler{
		output: w,
		fd:     fd,
		level:  level,
	}
}

func (c *ctlHandler) Enabled(_ context.Context, level Level) bool {
	return level >= c.level
}

func formatValue(val slog.Value) string {
	switch val.Kind() {
	case slog.KindString:
		return quoteIfNeed(val.String())
	case slog.KindDuration:
		return format.HumanDuration(val.Duration())
	default:
		switch x := val.Any().(type) {
		case error:
			return quoteIfNeed(x.Error())
		case fmt.Stringer:
			return quoteIfNeed(x.String())
		default:
			v, err := json.Marshal(x)
			if err == nil {
				return string(v)
			}
			return quoteIfNeed(val.String())
		}
	}
}

func (c *ctlHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Level < c.level {
		return nil
	}

	attrs := make([]string, 0, r.NumAttrs()+1)
	r.Attrs(func(attr slog.Attr) bool {
		value := formatValue(attr.Value)
		if len(c.groups) == 0 {
			attrs = append(attrs, attr.Key+"="+value)
		} else {
			attrs = append(attrs, strings.Join(append(c.groups, attr.Key), ".")+"="+value)
		}
		return true
	})

	if c.attrsStr == nil {
		attrs := make([]string, 0, len(c.attrs))
		for i := len(c.attrs) - 1; i >= 0; i-- {
			attr := c.attrs[i]
			attrs = append(attrs, attr.Key+"="+formatValue(attr.Value))
		}
		attrsStr := strings.Join(attrs, " ")
		c.attrsStr = &attrsStr
	}

	if c.attrsStr != nil {
		attrs = append(attrs, *c.attrsStr)
	}

	attrsStr := ""
	if len(attrs) != 0 {
		attrsStr = strings.Join(attrs, " ")
	}

	var termWidth int
	if c.fd != 0 {
		termWidth, _, _ = term.GetSize(c.fd)
	}
	log := formatLog(r.Message, attrsStr, r.Level, termWidth)
	_, err := io.WriteString(c.output, log)
	return err
}

func formatLog(msg string, attrs string, level Level, termWidth int) string {
	if attrs == "" {
		if level != LevelInfo {
			levelStr := level.String()
			c, ok := levelColor[strings.SplitN(levelStr, "+", 2)[0]]
			if ok {
				msg = c.renderer + " " + msg
			}
		}
		return fmt.Sprintf("%s\n", msg)
	}

	msgWidth := monospace.String(msg)
	if level != LevelInfo {
		levelStr := level.String()
		c, ok := levelColor[strings.SplitN(levelStr, "+", 2)[0]]
		if ok {
			msg = c.renderer + " " + msg
			msgWidth += c.width + 1
		}
	}
	if termWidth > msgWidth {
		return fmt.Sprintf("%s %*s\n", msg, termWidth-msgWidth-1, attrs)
	}

	return fmt.Sprintf("%s %s\n", msg, attrs)
}

type color struct {
	renderer string
	width    int
}

func newColour(c ctc.Color, msg string) color {
	return color{
		renderer: fmt.Sprintf("%s%s%s", c, msg, ctc.Reset),
		width:    monospace.String(msg),
	}
}

var levelColor = map[string]color{
	LevelError.String(): newColour(ctc.ForegroundRed, LevelError.String()),
	LevelWarn.String():  newColour(ctc.ForegroundYellow, LevelWarn.String()),
	LevelDebug.String(): newColour(ctc.ForegroundCyan, LevelDebug.String()),
}

func (c *ctlHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, 0, len(c.attrs)+len(attrs))
	newAttrs = append(newAttrs, c.attrs...)
	if len(c.groups) == 0 {
		newAttrs = append(newAttrs, attrs...)
	} else {
		for _, attr := range attrs {
			newAttrs = append(newAttrs, slog.Attr{
				Key:   strings.Join(append(c.groups, attr.Key), "."),
				Value: attr.Value,
			})
		}
	}
	return &ctlHandler{
		fd:     c.fd,
		level:  c.level,
		output: c.output,
		attrs:  newAttrs,
		groups: c.groups,
	}
}

func (c *ctlHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, 0, len(c.groups)+1)
	newGroups = append(newGroups, c.groups...)
	newGroups = append(newGroups, name)
	return &ctlHandler{
		fd:     c.fd,
		level:  c.level,
		output: c.output,
		attrs:  c.attrs,
		groups: newGroups,
	}
}

// quoteIfNeed returns wrap it in double quotes if the string contains characters other than letters and digits,
// otherwise return the original string
func quoteIfNeed(s string) string {
	for _, c := range s {
		if !unicode.Is(quoteRangeTable, c) {
			return strconv.Quote(s)
		}
	}
	return s
}

var quoteRangeTable = &unicode.RangeTable{
	R16: []unicode.Range16{
		{'-', '/', 1}, // '-' '.' '/'
		{'0', '9', 1},
		{':', ':', 1},
		{'A', 'Z', 1},
		{'_', '_', 1},
		{'a', 'z', 1},
	},
}

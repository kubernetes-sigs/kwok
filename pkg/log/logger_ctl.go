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
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/wzshiming/ctc"
	//nolint:depguard
	"golang.org/x/exp/slog"
	"golang.org/x/term"

	"sigs.k8s.io/kwok/pkg/utils/format"
)

type ctlHandler struct {
	level    slog.Level
	output   io.Writer
	attrs    []slog.Attr
	attrsStr *string
	groups   []string
	fd       int
}

func newCtlHandler(w io.Writer, fd int, level slog.Level) *ctlHandler {
	return &ctlHandler{
		output: w,
		fd:     fd,
		level:  level,
	}
}

func (c *ctlHandler) Enabled(_ context.Context, level slog.Level) bool {
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

	if c.attrsStr == nil {
		attrs := make([]string, 0, len(c.attrs))
		for _, attr := range c.attrs {
			attrs = append(attrs, attr.Key+"="+formatValue(attr.Value))
		}
		attrsStr := strings.Join(attrs, " ")
		c.attrsStr = &attrsStr
	}

	attrs := make([]string, 0, r.NumAttrs()+1)
	if c.attrsStr != nil {
		attrs = append(attrs, *c.attrsStr)
	}
	r.Attrs(func(attr slog.Attr) bool {
		value := formatValue(attr.Value)
		if len(c.groups) == 0 {
			attrs = append(attrs, attr.Key+"="+value)
		} else {
			attrs = append(attrs, strings.Join(append(c.groups, attr.Key), ".")+"="+value)
		}
		return true
	})

	attrsStr := ""
	if len(attrs) != 0 {
		attrsStr = strings.Join(attrs, " ")
	}

	msg := r.Message
	var err error
	if attrsStr == "" {
		if r.Level != slog.LevelInfo {
			levelStr := r.Level.String()
			c, ok := levelColor[strings.SplitN(levelStr, "+", 2)[0]]
			if ok {
				msg = c.renderer + " " + msg
			}
		}
		_, err = fmt.Fprintf(c.output, "%s\n", msg)
	} else {
		msgWidth := stringWidth(msg)
		if r.Level != slog.LevelInfo {
			levelStr := r.Level.String()
			c, ok := levelColor[strings.SplitN(levelStr, "+", 2)[0]]
			if ok {
				msg = c.renderer + " " + msg
				msgWidth += c.width + 1
			}
		}
		termWidth, _, _ := term.GetSize(c.fd)
		if termWidth > msgWidth {
			_, err = fmt.Fprintf(c.output, "%s%*s\n", msg, termWidth-1-msgWidth, attrsStr)
		} else {
			_, err = fmt.Fprintf(c.output, "%s%s\n", msg, attrsStr)
		}
	}
	return err
}

type color struct {
	renderer string
	width    int
}

func newColour(c ctc.Color, msg string) color {
	return color{
		renderer: fmt.Sprintf("%s%s%s", c, msg, ctc.Reset),
		width:    stringWidth(msg),
	}
}

var levelColor = map[string]color{
	slog.LevelError.String(): newColour(ctc.ForegroundRed, slog.LevelError.String()),
	slog.LevelWarn.String():  newColour(ctc.ForegroundYellow, slog.LevelWarn.String()),
	slog.LevelDebug.String(): newColour(ctc.ForegroundCyan, slog.LevelDebug.String()),
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

func stringWidth(str string) int {
	n := 0
	for _, r := range str {
		n += runeWidth(r)
	}
	return n
}

func runeWidth(r rune) int {
	switch {
	case r == utf8.RuneError || r < '\x20':
		return 0
	case '\x20' <= r && r < '\u2000':
		return 1
	case '\u2000' <= r && r < '\uFF61':
		return 2
	case '\uFF61' <= r && r < '\uFFA0':
		return 1
	case '\uFFA0' <= r:
		return 2
	}
	return 0
}

// quoteIfNeed returns wrap it in double quotes if the string contains characters other than letters and digits,
// otherwise return the original string
func quoteIfNeed(s string) string {
	for _, c := range s {
		if !unicode.In(c, unicode.Letter, unicode.Digit) {
			return strconv.Quote(s)
		}
	}
	return s
}

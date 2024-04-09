/*
Copyright 2024 The Kubernetes Authors.

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

package progressbar

import (
	"fmt"
	"strings"
	"time"

	"github.com/wzshiming/ctc"

	"sigs.k8s.io/kwok/pkg/utils/monospace"
)

func formatProgress(name string, width uint64, current uint64, total uint64, elapsed time.Duration) string {
	var per string
	if current == total {
		per = fmt.Sprintf("size=%s", formatBytes(total))
	} else {
		per = fmt.Sprintf("size=%s/%s", formatBytes(current), formatBytes(total))
	}

	e := elapsed.Truncate(time.Second)
	if e != 0 || current != total {
		widePer := fmt.Sprintf("%s speed=%s elapsed=%s", per, formatSpeed(current, elapsed), e)
		if len(widePer) < int(width)-1 {
			per = widePer
		}
	}

	info := formatInfo(width, name, per)
	index := calculateSplitIndex(info, width, current, total)
	if index != 0 {
		info = formatBar(info, index)
	}
	return info
}

func formatInfo(max uint64, preInfo, postInfo string) string {
	preInfoWidth := monospace.String(preInfo)
	postInfoWidth := monospace.String(postInfo)
	infoWidth := preInfoWidth + postInfoWidth
	if infoWidth >= int(max) {
		preInfoWidth = int(max) - postInfoWidth - 1
		preInfo = monospace.Shorten(preInfo, preInfoWidth)
	}
	return strings.Join([]string{
		preInfo,
		strings.Repeat(" ", int(max)-preInfoWidth-postInfoWidth),
		postInfo,
	}, "")
}

func calculateSplitIndex(info string, width uint64, current uint64, total uint64) uint64 {
	count := current * width / total
	if count == width {
		return 0
	}
	off := 0
	for i, ch := range info {
		off += monospace.String(string([]rune{ch}))
		if off > int(count) {
			return uint64(i)
		}
	}
	return 0
}

func formatBar(info string, index uint64) string {
	infoRunes := []rune(info)
	return strings.Join([]string{
		resetColor,
		negativeColor,
		string(infoRunes[:index]),
		resetColor,
		string(infoRunes[index:]),
	}, "")
}

var (
	resetColor    = ctc.Reset.String()
	negativeColor = ctc.Negative.String()
)

func formatSpeed(size uint64, elapsed time.Duration) string {
	second := elapsed.Seconds()
	if second < 1 {
		return formatBytes(size) + "/s"
	}
	return formatBytes(uint64(float64(size)/second)) + "/s"
}

var (
	binaryAbbrs = []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB", "YiB"}
)

func formatBytes(size uint64) string {
	s, u := getSizeAndUnit(float64(size), 1024, binaryAbbrs)
	return fmt.Sprintf("%.0f%s", s, u)
}

func getSizeAndUnit(size float64, base float64, abbrs []string) (float64, string) {
	i := 0
	unitsLimit := len(abbrs) - 1
	for size >= base && i < unitsLimit {
		size /= base
		i++
	}
	return size, abbrs[i]
}

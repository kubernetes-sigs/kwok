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

package width

import (
	"unicode/utf8"
)

// Width returns the width of a string in monospace font.
func Width(str string) int {
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

// Shorten returns a shortened string to fit the given width in monospace font.
func Shorten(str string, max int) string {
	if Width(str) <= max {
		return str
	}

	runes := []rune(str)
	begin := 0
	end := len(runes) - 1
	w := 0
	for i := 0; i < len(runes)/2; i++ {
		w += runeWidth(runes[begin])
		if w >= max-2 {
			break
		}
		begin++

		w += runeWidth(runes[end])
		if w >= max-2 {
			break
		}
		end--
	}

	return string(append(runes[:begin], append([]rune("..."), runes[end+1:]...)...))
}

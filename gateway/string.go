package gateway

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type byLengthAndContent []string

func (s byLengthAndContent) Len() int {
	return len(s)
}

func (s byLengthAndContent) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byLengthAndContent) Less(i, j int) bool {
	if len(s[i]) == len(s[j]) {
		return s[i] < s[j]
	}

	return len(s[i]) > len(s[j])
}

func sortBifrostVariables(slice []string) {
	sort.Sort(byLengthAndContent(slice))
}

func parseVariables(content string) []string {
	variables := reIsVariable.FindAllString(content, -1)
	sortBifrostVariables(variables)
	return variables
}

func escapeString(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		c := s[i]
		if c < utf8.RuneSelf {
			// ASCII 字符
			switch c {
			case '\\', '"':
				b.WriteByte('\\')
				b.WriteByte(c)
			case '\n':
				b.WriteString("\\n")
			case '\r':
				b.WriteString("\\r")
			case '\t':
				b.WriteString("\\t")
			case '\b':
				b.WriteString("\\b")
			case '\f':
				b.WriteString("\\f")
			default:
				if c < 0x20 {
					// 控制字符，转义为 \u00XX
					fmt.Fprintf(&b, "\\u%04x", c)
				} else {
					b.WriteByte(c)
				}
			}
			i++
		} else {
			// 非 ASCII 字符
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 {
				b.WriteString("\uFFFD")
				i++
			} else {
				b.WriteRune(r)
				i += size
			}
		}
	}
	return b.String()
}

package accesslog

import (
	"regexp"
	"sort"
	"unsafe"
)

var (
	reIsVariable = regexp.MustCompile(`\$\w+(-\w+)*`)
	questionByte = []byte{byte('?')}
)

func b2s(b []byte) string {
	return unsafe.String(unsafe.SliceData(b), len(b))
}

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

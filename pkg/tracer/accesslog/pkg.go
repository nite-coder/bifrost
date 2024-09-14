package accesslog

import (
	"regexp"
	"sort"
)

var (
	reIsVariable    = regexp.MustCompile(`\$\w+(-\w+)*`)
	questionByte    = []byte{byte('?')}
	grpcContentType = []byte("application/grpc")
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

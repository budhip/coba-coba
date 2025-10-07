package monitoring

import (
	"regexp"
	"strings"
)

// reFuncName regex pattern to capture package, receiver, and method names
var reFuncName = regexp.MustCompile(`(?:[^/]+/)*([^./]+)\.(?:\(?\*?([^.)]+)\)?\.)?(.+)$`)

func getSegmentName(fullFuncName string) string {
	matches := reFuncName.FindStringSubmatch(fullFuncName)
	if len(matches) < 4 {
		return fullFuncName
	}

	packageName := matches[1]
	receiver := matches[2]
	methodName := matches[3]

	var result []string
	if packageName != "" {
		result = append(result, packageName)
	}
	if receiver != "" {
		result = append(result, receiver)
	}
	if methodName != "" {
		result = append(result, methodName)
	}

	return strings.Join(result, ".")
}

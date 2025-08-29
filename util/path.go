package util

import "strings"

func StripLeadingSlashes(path string) (string, bool) {
	if path[0] != '/' {
		return path, false
	}
	return strings.TrimPrefix(path, "/"), true
}

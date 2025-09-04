package util

import "strings"

func StripLeadingSlashes(path string) string {
	return strings.TrimPrefix(path, "/")
}

package util

var version = "dev"
var gitrev = "unknown"

func GetVersion() string {
	return version
}

func GetGitRev() string {
	return gitrev
}

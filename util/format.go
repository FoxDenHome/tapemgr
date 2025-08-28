package util

import "strconv"

const (
	_          = iota
	KB float64 = 1 << (10 * iota)
	MB
	GB
	TB
	PB
)

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 2, 64)
}

func FormatSize(size int64) string {
	sz := float64(size)

	switch {
	case sz >= PB:
		return formatFloat(sz/PB) + " PB"
	case sz >= TB:
		return formatFloat(sz/TB) + " TB"
	case sz >= GB:
		return formatFloat(sz/GB) + " GB"
	case sz >= MB:
		return formatFloat(sz/MB) + " MB"
	case sz >= KB:
		return formatFloat(sz/KB) + " KB"
	default:
		return formatFloat(sz) + " B"
	}
}

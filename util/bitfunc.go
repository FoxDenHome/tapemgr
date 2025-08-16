package util

func BoolToFlag(val bool, pos uint8) uint8 {
	if val {
		return 1 << pos
	}
	return 0
}

func FlagToBool(flag uint8, pos uint8) bool {
	return (flag & (1 << pos)) != 0
}

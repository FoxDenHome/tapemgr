package scsi

func boolToFlag(val bool, pos uint8) uint8 {
	if val {
		return 1 << pos
	}
	return 0
}

func flagToBool(flag uint8, pos uint8) bool {
	return (flag & (1 << pos)) != 0
}

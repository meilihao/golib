package misc

func IsInInts(s int, list []int) bool {
	if len(list) == 0 {
		return false
	}

	for i := range list {
		if list[i] == s {
			return true
		}
	}

	return false
}

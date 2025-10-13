package misc

func IsInInts[T int | int64 | uint64](s T, list []T) bool {
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

func MaxInt[T int | int64 | uint64](a, b T) T {
	if a >= b {
		return a
	} else {
		return b
	}
}

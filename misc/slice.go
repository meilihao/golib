package misc

import "strings"

func IsInStrings(s string, list []string) bool {
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

func MatchPrefixSlice(name string, ls []string) bool {
	if len(ls) == 0 {
		return false
	}

	for i := range ls {
		if strings.HasPrefix(name, ls[i]) {
			return true
		}
	}

	return false
}

package misc

import "strings"

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

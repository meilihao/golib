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

func ExcludeStrings(in, ex []string) []string {
	if len(in) == 0 || len(ex) == 0 {
		return in
	}

	ls := make([]string, 0)
	for _, v := range in {
		if IsInStrings(v, ex) {
			continue
		}
		ls = append(ls, v)
	}
	return ls
}

func IsIncludedAnyStrings(s string, list []string) bool {
	if len(list) == 0 {
		return false
	}

	for i := range list {
		if strings.Contains(s, list[i]) {
			return true
		}
	}

	return false
}

func IsIncludedAnyStringsWithF(f func(string) string, s string, list []string) bool {
	if len(list) == 0 {
		return false
	}

	for i := range list {
		if strings.Contains(f(s), f(list[i])) {
			return true
		}
	}

	return false
}

func IsIncludedAnyStringsWithMatch(s string, list []string) string {
	if len(list) == 0 {
		return ""
	}

	for i := range list {
		if strings.Contains(s, list[i]) {
			return list[i]
		}
	}

	return ""
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

func IsAllNumbers(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

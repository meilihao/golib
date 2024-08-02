package misc

import "sort"

func Set(m map[string]struct{}) []string {
	ls := make([]string, 0, len(m))

	for k := range m {
		ls = append(ls, k)
	}

	sort.Strings(ls)
	return ls
}

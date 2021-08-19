package convert

// Strings2Interfaces []string -> []interface{}
func Strings2Interfaces(ss []string) []interface{} {
	ls := make([]interface{}, len(ss))

	for i := range ss {
		ls[i] = ss[i]
	}

	return ls
}

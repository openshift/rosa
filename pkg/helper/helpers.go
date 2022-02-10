package helper

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func SliceToMap(s []string) map[string]bool {
	m := make(map[string]bool)

	for _, v := range s {
		m[v] = true
	}

	return m
}

// RemoveStrFromSlice removes one occurrence of 'str' from the 's' slice if exists.
func RemoveStrFromSlice(s []string, str string) []string {
	for i, v := range s {
		if v == str {
			return append(s[:i], s[i+1:]...)
		}
	}

	return s
}

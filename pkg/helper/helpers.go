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

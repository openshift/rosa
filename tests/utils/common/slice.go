package common

func RemoveFromStringSlice(slice []string, value string) []string {
	var newSlice []string
	for _, v := range slice {
		if v != value {
			newSlice = append(newSlice, v)
		}
	}
	return newSlice
}

func SliceContains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func AppendToStringSliceIfNotExist(slice []string, value string) []string {
	if !SliceContains(slice, value) {
		slice = append(slice, value)
	}
	return slice
}

func UniqueStringValues(slice []string) []string {
	keys := make(map[string]bool)
	var uniq []string
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			uniq = append(uniq, entry)
		}
	}
	return uniq
}

package cache

import (
	"fmt"
	"os"
	"reflect"
)

func ConvertToStringSlice(slice interface{}) ([]string, bool, error) {
	values := reflect.ValueOf(slice)
	if values.Kind() != reflect.Slice {
		return nil, false, fmt.Errorf("input is not a slice")
	}

	var extractedStrings []string
	for i := 0; i < values.Len(); i++ {
		val := values.Index(i)
		switch val.Kind() {
		case reflect.String:
			extractedStrings = append(extractedStrings, val.String())
		case reflect.Interface:
			val = reflect.ValueOf(val.Interface())
			if val.Kind() == reflect.String {
				extractedStrings = append(extractedStrings, val.String())
			}
		case reflect.Invalid:
			// skip elements with unknown or unsupported kind
		default:
			return nil, false, fmt.Errorf("unsupported kind: %s", val.Kind().String())
		}
	}
	return extractedStrings, true, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

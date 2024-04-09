package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

func uniquePath(path []string) []string {
	keys := make(map[string]int)
	uniPath := make([]string, 0)

	for _, p := range path {
		if p == "" {
			p = "."
		}
		keys[p] = 1
	}

	for element := range keys {
		uniPath = append(uniPath, element)
	}

	sort.Strings(uniPath)

	return uniPath
}

func isExecutable(file string) (bool, error) {
	info, err := os.Stat(file)
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "windows" {
		fileExt := strings.ToLower(filepath.Ext(file))

		switch fileExt {
		case ".bat", ".cmd", ".com", ".exe", ".ps1":
			return true, nil
		}
		return false, nil
	}

	if m := info.Mode(); !m.IsDir() && m&0111 != 0 {
		return true, nil
	}

	return false, nil
}

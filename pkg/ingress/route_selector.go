package ingress

import (
	"fmt"
	"strings"
)

func GetRouteSelector(labelMatches string) (map[string]string, error) {
	routeSelectors := make(map[string]string)

	if labelMatches == "" {
		return routeSelectors, nil
	}

	for _, labelMatch := range strings.Split(labelMatches, ",") {
		if !strings.Contains(labelMatch, "=") {
			return nil, fmt.Errorf("Expected key=value format for label-match")
		}
		tokens := strings.Split(labelMatch, "=")
		routeSelectors[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
	}

	return routeSelectors, nil
}

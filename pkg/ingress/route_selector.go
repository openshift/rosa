package ingress

import (
	"fmt"
	"strings"
)

func GetRouteSelector(labelMatches string) (map[string]string, error) {
	if labelMatches == "" {
		return map[string]string{}, nil
	}
	routeSelectors := map[string]string{}
	for _, labelMatch := range strings.Split(labelMatches, ",") {
		if !strings.Contains(labelMatch, "=") {
			return nil, fmt.Errorf("Expected key=value format for label-match")
		}
		tokens := strings.Split(labelMatch, "=")
		routeSelectors[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
	}

	return routeSelectors, nil
}

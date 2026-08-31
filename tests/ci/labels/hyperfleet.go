package labels

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

type hyperfleetLabels struct {
	Validated Labels
}

var Hyperfleet = initHyperfleet()

func initHyperfleet() *hyperfleetLabels {
	hLabels := new(hyperfleetLabels)
	hLabels.Validated = Label("hyperfleet-validated")

	return hLabels
}

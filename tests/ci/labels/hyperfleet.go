package labels

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

type hyperfleetLabels struct {
	Validated  Labels
	Sanity     Labels
	InProgress Labels
}

var Hyperfleet = initHyperfleet()

func initHyperfleet() *hyperfleetLabels {
	hLabels := new(hyperfleetLabels)
	hLabels.Validated = Label("hyperfleet-validated")
	hLabels.Sanity = Label("hyperfleet-sanity")
	hLabels.InProgress = Label("hyperfleet-inprog")
	return hLabels
}

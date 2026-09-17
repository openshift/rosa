package labels

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

type hyperfleetLabels struct {
	Validated     Labels
	Sanity        Labels
	NotApplicable Labels
	Deferred      Labels
}

var Hyperfleet = initHyperfleet()

func initHyperfleet() *hyperfleetLabels {
	hLabels := new(hyperfleetLabels)
	hLabels.Validated = Label("hyperfleet-validated")
	hLabels.Sanity = Label("hyperfleet-sanity")
	hLabels.NotApplicable = Label("hyperfleet-na")
	hLabels.Deferred = Label("hyperfleet-deferred")

	return hLabels
}

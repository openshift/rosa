package labels

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

type hyperfleetLabels struct {
	Validated     Labels
	Sanity        Labels
	InProgress    Labels
	One           Labels
	Deferred      Labels
	NotApplicable Labels
}

var Hyperfleet = initHyperfleet()

func initHyperfleet() *hyperfleetLabels {
	hLabels := new(hyperfleetLabels)
	hLabels.Validated = Label("hyperfleet-validated")
	hLabels.Sanity = Label("hyperfleet-sanity")
	hLabels.InProgress = Label("hyperfleet-inprog")
	hLabels.One = Label("hyperfleet-one")
	hLabels.Deferred = Label("hyperfleet-deferred")
	hLabels.NotApplicable = Label("hyperfleet-na")

	return hLabels
}

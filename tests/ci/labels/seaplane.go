// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

type seaplaneLabels struct {
	Validated     Labels
	Sanity        Labels
	InProgress    Labels
	One           Labels
	Deferred      Labels
	NotApplicable Labels
}

var Seaplane = initSeaplane()

func initSeaplane() *seaplaneLabels {
	hLabels := new(seaplaneLabels)
	hLabels.Validated = Label("seaplane-validated")
	hLabels.Sanity = Label("seaplane-sanity")
	hLabels.InProgress = Label("seaplane-inprog")
	hLabels.One = Label("seaplane-one")
	hLabels.Deferred = Label("seaplane-deferred")
	hLabels.NotApplicable = Label("seaplane-na")

	return hLabels
}

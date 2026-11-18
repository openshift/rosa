// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package labels

import (
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
)

type seaplaneLabels struct {
	Validated     Labels
	Shared        Labels
	Sanity        Labels
	InProgress    Labels
	One           Labels
	Deferred      Labels
	NotApplicable Labels
	KnownIssue    Labels
	Profile       *seaplaneProfileLabels
}

// seaplaneProfileLabels scope a spec to lifecycle profiles. A spec without any
// profile label runs on every profile; a spec with one or more profile labels
// runs only on the matching TEST_PROFILE values.
type seaplaneProfileLabels struct {
	Basic               Labels
	FIPS                Labels
	Advanced            Labels
	Private             Labels
	NoCNI               Labels
	DisableCapabilities Labels
}

var Seaplane = initSeaplane()

func initSeaplane() *seaplaneLabels {
	hLabels := new(seaplaneLabels)
	hLabels.Validated = Label("seaplane-validated")
	hLabels.Shared = Label("seaplane-shared")
	hLabels.Sanity = Label("seaplane-sanity")
	hLabels.InProgress = Label("seaplane-inprog")
	hLabels.One = Label("seaplane-one")
	hLabels.Deferred = Label("seaplane-deferred")
	hLabels.NotApplicable = Label("seaplane-na")
	hLabels.KnownIssue = Label("seaplane-known-issue")

	hLabels.Profile = new(seaplaneProfileLabels)
	hLabels.Profile.Basic = Label("seaplane-profile-basic")
	hLabels.Profile.FIPS = Label("seaplane-profile-fips")
	hLabels.Profile.Advanced = Label("seaplane-profile-advanced")
	hLabels.Profile.Private = Label("seaplane-profile-private")
	hLabels.Profile.NoCNI = Label("seaplane-profile-nocni")
	hLabels.Profile.DisableCapabilities = Label("seaplane-profile-disable-capabilities")

	return hLabels
}

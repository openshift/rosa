package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

// Pre check
var E2ECommit = Label("e2e-commit")

// Report portal
var E2EReport = Label("e2e-report")

// Test
// The lables is always defined on each test case.
type runtimeLabels struct {
	// Test cases based on a cluster created by profiles.
	Day1        Labels
	Day1Post    Labels
	Day2        Labels
	Upgrade     Labels
	Destructive Labels
	Destroy     Labels
	DestroyPost Labels

	// Test cases beyond the cluster created by profiles.
	Day1Supplemental Labels
	Day1Negative     Labels
	OCMResources     Labels
}

var Runtime = initRuntime()

func initRuntime() *runtimeLabels {
	var rLabels = new(runtimeLabels)
	rLabels.Day1 = Label("day1")
	rLabels.Day1Post = Label("day1-post")
	rLabels.Day2 = Label("day2")
	rLabels.Upgrade = Label("upgrade")
	rLabels.Destructive = Label("destructive")
	rLabels.Destroy = Label("feature-destroy")
	rLabels.DestroyPost = Label("feature-destroy-post")

	rLabels.Day1Supplemental = Label("feature-day1-supplemental")
	rLabels.OCMResources = Label("feature-ocm-resources")
	rLabels.Day1Negative = Label("feature-day1-negative")

	return rLabels
}

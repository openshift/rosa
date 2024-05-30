package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

// Cluster Type
var NonClassicCluster = Label("NonClassicCluster")

var NonHCPCluster = Label("NonHCPCluster")

// exclude
var Exclude = Label("Exclude")

var MigrationToVerify = Label("MigrationToVerify")

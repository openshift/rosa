package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

var FeatureCluster = Label("feature-cluster")
var FeatureRoles = Label("feature-roles")
var FeatureIDP = Label("feature-idp")
var FeatureIngress = Label("feature-ingress")
var FeatureKubeletConfig = Label("feature-kubeletconfig")
var FeatureMachinepool = Label("feature-machinepool")
var FeatureNetworkVerifier = Label("feature-networkverifier")
var FeatureNodePool = Label("feature-nodepool")
var FeatureOidcConfig = Label("feature-oidcconfig")
var FeatureOidcProvider = Label("feature-oidcprovider")
var FeatureRegion = Label("feature-region")
var FeatureUser = Label("feature-user")

var FeatureCLI = Label("feature-cli") // For tests related to command line directly

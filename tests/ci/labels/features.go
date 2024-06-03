package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

/* The feature labels define which group the test case belongs to.
The feature label is always defined at the first `Description`.
We treat the major subcommands in rosa CLI as features. The label items are from `rosa create -h` -> `rosa list -h` -> `rosa -h`.
*/
// feature labels
type featureLabels struct {
	AccountRoles         Labels
	Addon                Labels
	Autoscaler           Labels
	Cluster              Labels
	BreakGlassCredential Labels
	ExternalAuthProvider Labels
	Gates                Labels
	IDP                  Labels
	Ingress              Labels
	InstanceTypes        Labels
	KubeletConfig        Labels
	Machinepool          Labels
	OCMRole              Labels
	OIDCConfig           Labels
	OIDCProvider         Labels
	OperatorRoles        Labels
	Regions              Labels
	Token                Labels
	TuningConfigs        Labels
	UserRole             Labels
	VerifyResources      Labels
	Version              Labels
}

var Feature = initFeatureLabels()

func initFeatureLabels() *featureLabels {
	var fLabels = new(featureLabels)
	fLabels.AccountRoles = Label("feature-account-roles")
	fLabels.Addon = Label("feature-addon")
	fLabels.Autoscaler = Label("feature-autoscaler")
	fLabels.Cluster = Label("feature-cluster")
	fLabels.BreakGlassCredential = Label("feature-break-glass-credential")
	fLabels.ExternalAuthProvider = Label("feature-external-auth-provider")
	fLabels.Gates = Label("feature-gates")
	fLabels.IDP = Label("feature-idp")
	fLabels.Ingress = Label("feature-ingress")
	fLabels.InstanceTypes = Label("feature-instance-types")
	fLabels.KubeletConfig = Label("feature-kubeletconfig")
	fLabels.Machinepool = Label("feature-machinepool")
	fLabels.OCMRole = Label("feature-ocm-role")
	fLabels.OIDCConfig = Label("feature-oidc-config")
	fLabels.OIDCProvider = Label("feature-oidc-provider")
	fLabels.OperatorRoles = Label("feature-operator-roles")
	fLabels.Token = Label("feature-token")
	fLabels.TuningConfigs = Label("feature-tuning-configs")
	fLabels.UserRole = Label("feature-user-role")
	fLabels.VerifyResources = Label("feature-verify-resources")
	fLabels.Version = Label("feature-version")

	return fLabels
}

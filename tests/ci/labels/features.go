package labels

import (
	"github.com/onsi/ginkgo/v2"
)

// feature labels
// The feature label is always defined at the first `Description`.
type featureLabels struct {
	AccountRoles         ginkgo.Labels
	Addon                ginkgo.Labels
	Autoscaler           ginkgo.Labels
	Cluster              ginkgo.Labels
	BreakGlassCredential ginkgo.Labels
	ExternalAuthProvider ginkgo.Labels
	Gates                ginkgo.Labels
	IAMServiceAccount    ginkgo.Labels
	ImageMirror          ginkgo.Labels
	IDP                  ginkgo.Labels
	Ingress              ginkgo.Labels
	InstanceTypes        ginkgo.Labels
	KubeletConfig        ginkgo.Labels
	Machinepool          ginkgo.Labels
	OCMRole              ginkgo.Labels
	OIDCConfig           ginkgo.Labels
	OIDCProvider         ginkgo.Labels
	OperatorRoles        ginkgo.Labels
	Policy               ginkgo.Labels
	Regions              ginkgo.Labels
	Token                ginkgo.Labels
	TuningConfigs        ginkgo.Labels
	UserRole             ginkgo.Labels
	NetworkResources     ginkgo.Labels
	VerifyResources      ginkgo.Labels
	Version              ginkgo.Labels
	Upgrade              ginkgo.Labels
	Config               ginkgo.Labels
	Hibernation          ginkgo.Labels
}

var Feature = initFeatureLabels()

func initFeatureLabels() *featureLabels {
	var fLabels = new(featureLabels)
	fLabels.AccountRoles = ginkgo.Label("feature-account-roles")
	fLabels.Addon = ginkgo.Label("feature-addon")
	fLabels.Autoscaler = ginkgo.Label("feature-autoscaler")
	fLabels.Cluster = ginkgo.Label("feature-cluster")
	fLabels.BreakGlassCredential = ginkgo.Label("feature-break-glass-credential")
	fLabels.ExternalAuthProvider = ginkgo.Label("feature-external-auth-provider")
	fLabels.Gates = ginkgo.Label("feature-gates")
	fLabels.IAMServiceAccount = ginkgo.Label("feature-iam-service-account")
	fLabels.ImageMirror = ginkgo.Label("feature-image-mirror")
	fLabels.IDP = ginkgo.Label("feature-idp")
	fLabels.Ingress = ginkgo.Label("feature-ingress")
	fLabels.InstanceTypes = ginkgo.Label("feature-instance-types")
	fLabels.KubeletConfig = ginkgo.Label("feature-kubeletconfig")
	fLabels.Machinepool = ginkgo.Label("feature-machinepool")
	fLabels.OCMRole = ginkgo.Label("feature-ocm-role")
	fLabels.OIDCConfig = ginkgo.Label("feature-oidc-config")
	fLabels.OIDCProvider = ginkgo.Label("feature-oidc-provider")
	fLabels.OperatorRoles = ginkgo.Label("feature-operator-roles")
	fLabels.Policy = ginkgo.Label("feature-policy")
	fLabels.Token = ginkgo.Label("feature-token")
	fLabels.TuningConfigs = ginkgo.Label("feature-tuning-configs")
	fLabels.UserRole = ginkgo.Label("feature-user-role")
	fLabels.VerifyResources = ginkgo.Label("feature-verify-resources")
	fLabels.Version = ginkgo.Label("feature-version")
	fLabels.Upgrade = ginkgo.Label("feature-upgrade")
	fLabels.Config = ginkgo.Label("feature-config")
	fLabels.Hibernation = ginkgo.Label("feature-hibernation")

	return fLabels
}

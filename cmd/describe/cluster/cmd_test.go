package cluster

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/ginkgo/v2/dsl/table"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	version string = "4.10.1"
	state   string = "running"
)

var (
	now                             = time.Now()
	expectEmptyCuster               = []byte(`{"displayName":"displayname","kind":"Cluster"}`)
	expectClusterWithNameAndIDValue = []byte(
		`{"displayName":"displayname","id":"bar","kind":"Cluster","name":"foo"}`)
	expectClusterWithExternalAuthConfig = []byte(
		`{"displayName":"displayname","external_auth_config":{"enabled":true},"kind":"Cluster"}`)
	expectClusterWithAap = []byte(
		`{"aws":{"additional_allowed_principals":["foobar"]},"displayName":"displayname","kind":"Cluster"}`)
	expectClusterWithNameAndValueAndUpgradeInformation = []byte(
		`{"displayName":"displayname","id":"bar","kind":"Cluster","name":"foo","scheduledUpgrade":{"nextRun":"` +
			now.Format("2006-01-02 15:04 MST") + `","state":"` + state + `","version":"` +
			version + `"}}`)
	expectEmptyClusterWithNameAndValueAndUpgradeInformation = []byte(
		`{"displayName":"displayname","kind":"Cluster","scheduledUpgrade":{"nextRun":"` +
			now.Format("2006-01-02 15:04 MST") + `","state":"` +
			state + `","version":"` +
			version + `"}}`)
	clusterWithNameAndID, emptyCluster, clusterWithExternalAuthConfig, clusterWithAap *cmv1.Cluster
	emptyUpgradePolicy, upgradePolicyWithVersionAndNextRun                            *cmv1.UpgradePolicy
	emptyUpgradeState, upgradePolicyWithState                                         *cmv1.UpgradePolicyState

	berr error
)
var _ = BeforeSuite(func() {
	clusterWithNameAndID, berr = cmv1.NewCluster().Name("foo").ID("bar").Build()
	Expect(berr).NotTo(HaveOccurred())
	emptyCluster, berr = cmv1.NewCluster().Build()
	Expect(berr).NotTo(HaveOccurred())
	externalAuthConfig := cmv1.NewExternalAuthConfig().Enabled(true)
	clusterWithExternalAuthConfig, berr = cmv1.NewCluster().ExternalAuthConfig(externalAuthConfig).Build()
	Expect(berr).NotTo(HaveOccurred())
	additionalAllowedPrincipals := cmv1.NewAWS().AdditionalAllowedPrincipals("foobar")
	clusterWithAap, berr = cmv1.NewCluster().AWS(additionalAllowedPrincipals).Build()
	Expect(berr).NotTo(HaveOccurred())
	emptyUpgradePolicy, berr = cmv1.NewUpgradePolicy().Build()
	Expect(berr).NotTo(HaveOccurred())
	emptyUpgradeState, berr = cmv1.NewUpgradePolicyState().Build()
	Expect(berr).NotTo(HaveOccurred())
	upgradePolicyWithVersionAndNextRun, berr = cmv1.NewUpgradePolicy().Version(version).NextRun(now).Build()
	Expect(berr).NotTo(HaveOccurred())
	upgradePolicyWithState, berr = cmv1.NewUpgradePolicyState().Value(cmv1.UpgradePolicyStateValue(state)).Build()
	Expect(berr).NotTo(HaveOccurred())

})
var _ = Describe("Cluster description", Ordered, func() {

	Context("when displaying clusters with output json", func() {

		DescribeTable("When displaying clusters with output json",
			printJson,
			Entry("Prints empty when all values are empty",
				func() *cmv1.Cluster { return emptyCluster },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectEmptyCuster, nil),

			Entry("Prints cluster information only",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster and upgrade information",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return upgradePolicyWithVersionAndNextRun },
				func() *cmv1.UpgradePolicyState { return upgradePolicyWithState },
				expectClusterWithNameAndValueAndUpgradeInformation, nil),

			Entry("Prints empty cluster with cluster information",
				func() *cmv1.Cluster { return emptyCluster },
				func() *cmv1.UpgradePolicy { return upgradePolicyWithVersionAndNextRun },
				func() *cmv1.UpgradePolicyState { return upgradePolicyWithState },
				expectEmptyClusterWithNameAndValueAndUpgradeInformation, nil),

			Entry("Prints cluster information only when no upgrade policy state is found",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return upgradePolicyWithVersionAndNextRun },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information only when no upgrade policy version and next run is found",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information only when upgrade policy is nil",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return nil },
				func() *cmv1.UpgradePolicyState { return emptyUpgradeState }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information only when upgrade policy state is nil",
				func() *cmv1.Cluster { return clusterWithNameAndID },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return nil }, expectClusterWithNameAndIDValue, nil),

			Entry("Prints cluster information with the external authentication provider",
				func() *cmv1.Cluster { return clusterWithExternalAuthConfig },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return nil }, expectClusterWithExternalAuthConfig, nil),

			Entry("Prints cluster information with the additional allowed principals",
				func() *cmv1.Cluster { return clusterWithAap },
				func() *cmv1.UpgradePolicy { return emptyUpgradePolicy },
				func() *cmv1.UpgradePolicyState { return nil }, expectClusterWithAap, nil),
		)
	})
})

func printJson(cluster func() *cmv1.Cluster,
	upgrade func() *cmv1.UpgradePolicy,
	state func() *cmv1.UpgradePolicyState,
	expected []byte,
	err error) {
	f, er := formatCluster(cluster(), upgrade(), state(), "displayname")
	if err != nil {
		Expect(er).To(Equal(err))
	}
	Expect(er).To(BeNil())
	v, er := json.Marshal(f)
	Expect(er).NotTo(HaveOccurred())
	Expect(v).To(Equal(expected))
}

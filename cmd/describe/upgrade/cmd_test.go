package upgrade

import (
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Describe upgrade", func() {
	Context("Format Hypershift upgrade", func() {
		It("Node pool upgrade is scheduled", func() {
			nowUTC := time.Now().UTC()
			upgradeState := cmv1.NewUpgradePolicyState().Value("scheduled").Description("Upgrade scheduled.")
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").
				State(upgradeState).NextRun(nowUTC).Build()
			Expect(err).To(BeNil())
			result := formatHypershiftUpgrade(npUpgradePolicy)
			Expect(result).To(Equal(
				fmt.Sprintf(
					`
ID:                                id1
Cluster ID:                        
Schedule Type:                     
Next Run:                          %s
Upgrade State:                     scheduled
State Message:                     Upgrade scheduled.

Version:                           4.12.19
`, nowUTC.Format("2006-01-02 15:04 MST"))))
		})
		It("Node pool upgrade is scheduled with a date", func() {
			format.TruncatedDiff = false
			nowUTC := time.Now().UTC()
			upgradeState := cmv1.NewUpgradePolicyState().Value("scheduled").Description("Upgrade scheduled.")
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").
				State(upgradeState).NextRun(nowUTC).Schedule(nowUTC.Format("2006-01-02 15:04 MST")).
				EnableMinorVersionUpgrades(true).Build()
			Expect(err).To(BeNil())
			result := formatHypershiftUpgrade(npUpgradePolicy)
			Expect(result).To(Equal(
				fmt.Sprintf(
					`
ID:                                id1
Cluster ID:                        
Schedule Type:                     
Next Run:                          %s
Upgrade State:                     scheduled
State Message:                     Upgrade scheduled.

Schedule At:                       %s

Enable minor version upgrades:     true

Version:                           4.12.19
`, nowUTC.Format("2006-01-02 15:04 MST"), nowUTC.Format("2006-01-02 15:04 MST"))))
		})
	})
	Context("Describe Classic Upgrades", func() {
		var testRuntime test.TestingRuntime
		var clusterID = "cluster1"
		var nodePoolID = "nodepool85"

		version41224 := cmv1.NewVersion().ID("openshift-v4.12.24").RawID("4.12.24").ReleaseImage("1").
			HREF("/api/clusters_mgmt/v1/versions/openshift-v4.12.24").Enabled(true).ChannelGroup("stable").
			ROSAEnabled(true).HostedControlPlaneEnabled(true).AvailableUpgrades("4.12.25", "4.12.26")
		nodePool, err := cmv1.NewNodePool().ID("workers").Replicas(2).AutoRepair(true).Version(version41224).Build()
		Expect(err).To(BeNil())

		upgradePolicies := make([]*cmv1.NodePoolUpgradePolicy, 0)
		upgradePolicies = append(upgradePolicies, buildNodePoolUpgradePolicy())

		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Upgrade policy found, no error", func() {
			args.nodePool = nodePoolID
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatNodePoolUpgradePolicyList(upgradePolicies)))
			err := describeClassicUpgrades(testRuntime.RosaRuntime, clusterID)
			Expect(err).To(BeNil())
		})
		It("Node pool upgrade is scheduled with a date", func() {
			format.TruncatedDiff = false
			nowUTC := time.Now().UTC()
			upgradeState, err := cmv1.NewUpgradePolicyState().Value("scheduled").Build()
			Expect(err).To(BeNil())
			npUpgradePolicy, err := cmv1.NewUpgradePolicy().ID("id1").ClusterID("id1").
				NextRun(nowUTC).Schedule(nowUTC.Format("2006-01-02 15:04 MST")).Version("4.12.25").Build()
			Expect(err).To(BeNil())
			result := formatClassicUpgrade(npUpgradePolicy, upgradeState)
			Expect(result).To(Equal(
				fmt.Sprintf(
					`
ID:                                id1
Cluster ID:                        id1
Next Run:                          %s
Upgrade State:                     scheduled

Schedule At:                       %s

Version:                           4.12.25
`, nowUTC.Format("2006-01-02 15:04 MST"), nowUTC.Format("2006-01-02 15:04 MST"))))
		})
	})

	Context("Describe Hypershift upgrade", func() {
		var testRuntime test.TestingRuntime
		var clusterID = "cluster1"
		var nodePoolID = "nodepool85"

		version41224 := cmv1.NewVersion().ID("openshift-v4.12.24").RawID("4.12.24").ReleaseImage("1").
			HREF("/api/clusters_mgmt/v1/versions/openshift-v4.12.24").Enabled(true).ChannelGroup("stable").
			ROSAEnabled(true).HostedControlPlaneEnabled(true).AvailableUpgrades("4.12.25", "4.12.26")
		nodePool, err := cmv1.NewNodePool().ID("workers").Replicas(2).AutoRepair(true).Version(version41224).Build()
		Expect(err).To(BeNil())

		upgradePolicies := make([]*cmv1.NodePoolUpgradePolicy, 0)
		upgradePolicies = append(upgradePolicies, buildNodePoolUpgradePolicy())

		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Upgrade policy found, no error", func() {
			args.nodePool = nodePoolID
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatNodePoolUpgradePolicyList(upgradePolicies)))
			err := describeHypershiftUpgrades(testRuntime.RosaRuntime, clusterID, nodePoolID)
			Expect(err).To(BeNil())
		})
		It("Error on the node pool response", func() {
			args.nodePool = nodePoolID
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			err := describeHypershiftUpgrades(testRuntime.RosaRuntime, clusterID, nodePoolID)
			Expect(err).ToNot(BeNil())
		})
	})
})

func buildNodePoolUpgradePolicy() *cmv1.NodePoolUpgradePolicy {
	t, err := time.Parse(time.RFC3339, "2023-06-02T12:30:00Z")
	Expect(err).To(BeNil())
	state := cmv1.NewUpgradePolicyState().Value(cmv1.UpgradePolicyStateValuePending)
	policy, err := cmv1.NewNodePoolUpgradePolicy().ScheduleType(cmv1.ScheduleTypeManual).
		UpgradeType(cmv1.UpgradeTypeNodePool).Version("4.12.25").State(state).NextRun(t).Build()
	Expect(err).To(BeNil())
	return policy
}

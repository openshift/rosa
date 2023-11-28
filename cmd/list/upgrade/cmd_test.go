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

const noUpgradeOutput = "There are no available upgrades for machine pool 'nodepool85'"
const ongoingUpgradeOutput = `VERSION  NOTES
4.12.26  recommended
4.12.25  pending for 2023-06-02 12:30 UTC
`
const upgradeAvailableOutput = `VERSION  NOTES
4.12.26  recommended
4.12.25  
`

var _ = Describe("List upgrade", func() {
	Context("Format scheduled upgrade", func() {
		It("Node pool upgrade is scheduled", func() {
			availableUpgrade := "4.12.19"
			nowUTC := time.Now().UTC()
			upgradeState := cmv1.NewUpgradePolicyState().Value("scheduled")
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").
				State(upgradeState).NextRun(nowUTC).Build()
			Expect(err).To(BeNil())
			notes := formatScheduledUpgradeHypershift(availableUpgrade, npUpgradePolicy)
			Expect(notes).To(Equal(fmt.Sprintf("scheduled for %s", nowUTC.Format("2006-01-02 15:04 MST"))))
		})
		It("Nothing scheduled for this node pool", func() {
			availableUpgrade := "4.12.18"
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").Build()
			Expect(err).To(BeNil())
			notes := formatScheduledUpgradeHypershift(availableUpgrade, npUpgradePolicy)
			Expect(notes).To(Equal(""))
		})
	})
	Context("Latest rev in minor", func() {
		It("Find latest minor", func() {
			currentVersion := "4.12.16"
			availableUpgrades := []string{"4.12.17", "4.12.18", "4.13.0"}
			latestRev := latestInCurrentMinor(currentVersion, availableUpgrades)
			Expect(latestRev).To(Equal("4.12.18"))
		})
		It("Only upgrades to a major available", func() {
			currentVersion := "4.12.16"
			availableUpgrades := []string{"4.13.0"}
			latestRev := latestInCurrentMinor(currentVersion, availableUpgrades)
			Expect(latestRev).To(Equal("4.12.16"))
		})
	})
	Context("List upgrades command", func() {
		var testRuntime test.TestingRuntime
		var nodePoolName = "nodepool85"

		mockClusterError, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateError)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})
		Expect(err).To(BeNil())
		var hypershiftClusterNotReady = test.FormatClusterList([]*cmv1.Cluster{mockClusterError})

		mockClusterReady, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
			c.Version(cmv1.NewVersion().RawID("4.12.26").ChannelGroup("stable").
				ID("4.12.26").Enabled(true).AvailableUpgrades("4.12.27"))
		})
		Expect(err).To(BeNil())
		var hypershiftClusterReady = test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		mockClassicCluster, err := test.MockOCMCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(false))
		})
		Expect(err).To(BeNil())
		var classicCluster = test.FormatClusterList([]*cmv1.Cluster{mockClassicCluster})

		versionNoUpgrades := cmv1.NewVersion().ID("openshift-v4.12.24").RawID("4.12.24").ReleaseImage("1").
			HREF("/api/clusters_mgmt/v1/versions/openshift-v4.12.24").Enabled(true).ChannelGroup("stable").
			ROSAEnabled(true).HostedControlPlaneEnabled(true)
		version41224 := cmv1.NewVersion().ID("openshift-v4.12.24").RawID("4.12.24").ReleaseImage("1").
			HREF("/api/clusters_mgmt/v1/versions/openshift-v4.12.24").Enabled(true).ChannelGroup("stable").
			ROSAEnabled(true).HostedControlPlaneEnabled(true).AvailableUpgrades("4.12.25", "4.12.26")
		nodePool, err := cmv1.NewNodePool().ID("workers").Replicas(2).AutoRepair(true).Version(version41224).Build()
		Expect(err).To(BeNil())
		nodePoolNoUpgrades, err := cmv1.NewNodePool().ID("workers").Replicas(2).AutoRepair(true).
			Version(versionNoUpgrades).Build()
		Expect(err).To(BeNil())
		emptyUpgradePolicies := make([]*cmv1.NodePoolUpgradePolicy, 0)

		upgradePolicies := make([]*cmv1.NodePoolUpgradePolicy, 0)
		upgradePolicies = append(upgradePolicies, buildNodePoolUpgradePolicy())

		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Fails if cluster is not hypershift and we are using hypershift specific flags", func() {
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(
				ContainSubstring("The '--machinepool' option is only supported for Hosted Control Planes"))
		})
		It("Fails if cluster is not ready", func() {
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterNotReady))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Cluster 'cluster1' is not yet ready"))
		})

		It("Cluster is ready and node pool no upgrade available", func() {
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			// A node pool
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePoolNoUpgrades)))
			// No existing policy upgrade
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatNodePoolUpgradePolicyList(emptyUpgradePolicies)))
			stdout, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(stdout).To(ContainSubstring(noUpgradeOutput))
			Expect(err).To(BeNil())
		})

		It("Cluster is ready and node pool can be upgraded with 2 upgrades available", func() {
			format.TruncatedDiff = true
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			// A node pool
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			// No existing policy upgrade
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatNodePoolUpgradePolicyList(emptyUpgradePolicies)))
			stdout, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(stdout).To(Equal(upgradeAvailableOutput))
			Expect(err).To(BeNil())
		})

		It("Cluster is ready and node pool can be upgraded with 2 upgrades available, 1 is scheduled", func() {
			format.TruncatedDiff = false
			args.nodePool = nodePoolName
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			// A node pool
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			// An existing policy upgrade
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatNodePoolUpgradePolicyList(upgradePolicies)))
			stdout, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(stdout).To(Equal(ongoingUpgradeOutput))
			Expect(err).To(BeNil())
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

package machinepool

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

const (
	scheduleTime        = "10:00"
	invalidScheduleDate = "25h December"
	validScheduleDate   = "2023-12-25"
	cronSchedule        = "* * * * *"
	invalidVersionError = `Expected a valid machine pool version: A valid version number must be specified
Valid versions: 4.12.26 4.12.25`
)

var _ = Describe("Upgrade machine pool", func() {
	Context("Upgrade machine pool command", func() {
		var testRuntime test.TestingRuntime
		var nodePoolName = "nodepool85"
		mockClusterError := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateError)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})
		hypershiftClusterNotReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterError})

		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady)
			c.Hypershift(cmv1.NewHypershift().Enabled(true))
		})

		hypershiftClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

		version41224 := cmv1.NewVersion().ID("openshift-v4.12.24").RawID("4.12.24").ReleaseImage("1").
			HREF("/api/clusters_mgmt/v1/versions/openshift-v4.12.24").Enabled(true).ChannelGroup("stable").
			ROSAEnabled(true).HostedControlPlaneEnabled(true).AvailableUpgrades("4.12.25", "4.12.26")
		nodePool, err := cmv1.NewNodePool().ID("workers").Replicas(2).AutoRepair(true).Version(version41224).Build()
		Expect(err).To(BeNil())

		upgradePolicies := make([]*cmv1.NodePoolUpgradePolicy, 0)
		upgradePolicies = append(upgradePolicies, buildNodePoolUpgradePolicy())
		nodePoolUpgradePolicy := test.FormatNodePoolUpgradePolicyList(upgradePolicies)

		noNodePoolUpgradePolicy := test.FormatNodePoolUpgradePolicyList([]*cmv1.NodePoolUpgradePolicy{})

		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Fails if we are using minor version flag for manual upgrades", func() {
			args.schedule = ""
			args.allowMinorVersionUpdates = true
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd, []string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("The '--allow-minor-version-upgrades' " +
				"option needs to be used with --schedule"))
		})
		It("Fails if we are mixing scheduling type flags", func() {
			args.schedule = cronSchedule
			args.scheduleDate = "31 Jan"
			args.allowMinorVersionUpdates = false
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd, []string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("The '--schedule-date' and '--schedule-time' " +
				"options are mutually exclusive with '--schedule'"))
		})
		It("Fails if we are mixing automatic scheduling and version flags", func() {
			args.schedule = cronSchedule
			args.scheduleDate = ""
			args.version = "4.13.0"
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd, []string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("The '--schedule' " +
				"option is mutually exclusive with '--version'"))
		})
		It("Fails if cluster is not ready", func() {
			args.schedule = ""
			args.version = ""
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterNotReady))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd, []string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Cluster 'cluster1' is not yet ready"))
		})
		It("Cluster is ready but node pool not found", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = validScheduleDate
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			err := runWithRuntime(testRuntime.RosaRuntime, Cmd, []string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				"Machine pool 'nodepool85' does not exist for hosted cluster 'cluster1'"))
		})
		It("Cluster is ready and there is a scheduled upgraded", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
			_, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(ContainSubstring(
				"WARN: There is already a scheduled upgrade to version 4.12.25 on 2023-08-07 15:22 UTC"))
		})
		It("Succeeds if cluster is ready and there is a scheduled upgraded", func() {
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
			_, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(ContainSubstring(
				"WARN: There is already a scheduled upgrade to version 4.12.25 on 2023-08-07 15:22 UTC"))
		})
		It("Fails if cluster is ready and there is no scheduled upgraded but schedule date is invalid", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = invalidScheduleDate
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(
				"schedule date should use the format 'yyyy-mm-dd'"))
			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(BeEmpty())
		})
		It("Fails if cluster is ready and there is no scheduled upgraded but a version not "+
			"in available upgrades is specified",
			func() {
				args.scheduleTime = scheduleTime
				args.scheduleDate = validScheduleDate
				Cmd.Flags().Set("version", "4.13.26")
				Cmd.Flags().Set("interactive", "false")
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
				testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
				stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
					Cmd, &[]string{nodePoolName})
				Expect(err).ToNot(BeNil())
				Expect(err.Error()).To(Equal(invalidVersionError))
				Expect(stderr).To(BeEmpty())
				Expect(stdout).To(BeEmpty())
			})
		It("Succeeds if cluster is ready and there is no scheduled upgraded and a version is specified", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = validScheduleDate
			Cmd.Flags().Set("version", "4.12.26")
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring(
				"Upgrade successfully scheduled for the machine pool 'nodepool85' on cluster 'cluster1"))
		})
		It("Succeeds if cluster is ready and there is no scheduled upgraded", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = validScheduleDate
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring(
				"Upgrade successfully scheduled for the machine pool 'nodepool85' on cluster 'cluster1"))
		})
		It("Cluster is ready and there is no scheduled upgraded but scheduling fails due to a BE error", func() {
			args.scheduleTime = scheduleTime
			args.scheduleDate = validScheduleDate
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusBadRequest, "an error"))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Failed to schedule upgrade for machine pool"))
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(BeEmpty())
		})
		It("Cluster is ready and with automatic scheduling but bad cron format", func() {
			args.scheduleTime = ""
			args.scheduleDate = ""
			args.version = ""
			// not a valid cron
			args.schedule = "* a"
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
			_, _, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("Schedule '* a' is not a valid cron expression"))
		})
		It("Cluster is ready and with automatic scheduling and good cron format -> success", func() {
			args.scheduleTime = ""
			args.scheduleDate = ""
			args.version = ""
			// not a valid cron
			args.schedule = cronSchedule
			Cmd.Flags().Set("interactive", "false")
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, test.FormatResource(nodePool)))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, noNodePoolUpgradePolicy))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, ""))
			stdout, stderr, err := test.RunWithOutputCaptureAndArgv(runWithRuntime, testRuntime.RosaRuntime,
				Cmd, &[]string{nodePoolName})
			Expect(err).To(BeNil())
			Expect(stderr).To(BeEmpty())
			Expect(stdout).To(ContainSubstring(
				"Upgrade successfully scheduled for the machine pool 'nodepool85' on cluster 'cluster1'"))
		})
	})
})

func buildNodePoolUpgradePolicy() *cmv1.NodePoolUpgradePolicy {
	t, err := time.Parse(time.RFC3339, "2023-08-07T15:22:00Z")
	Expect(err).To(BeNil())
	state := cmv1.NewUpgradePolicyState().Value(cmv1.UpgradePolicyStateValueScheduled)
	policy, err := cmv1.NewNodePoolUpgradePolicy().ScheduleType(cmv1.ScheduleTypeManual).
		UpgradeType(cmv1.UpgradeTypeNodePool).Version("4.12.25").State(state).NextRun(t).Build()
	Expect(err).To(BeNil())
	return policy
}

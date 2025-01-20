package cluster

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Upgrade", Ordered, func() {
	var testRuntime test.TestingRuntime

	const cronSchedule = "* * * * *"
	const timeSchedule = "10:00"
	const dateSchedule = "2023-06-01"
	var emptyClusterList = test.FormatClusterList([]*cmv1.Cluster{})
	version4130 := cmv1.NewVersion().ID("openshift-v4.13.0").RawID("4.13.0").ReleaseImage("1").
		HREF("/api/clusters_mgmt/v1/versions/openshift-v4.13.0").Enabled(true).ChannelGroup("stable").
		ROSAEnabled(true).HostedControlPlaneEnabled(true)

	mockClusterError := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateError)
		c.Hypershift(cmv1.NewHypershift().Enabled(true))
	})
	var hypershiftClusterNotReady = test.FormatClusterList([]*cmv1.Cluster{mockClusterError})

	mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateReady)
		c.Hypershift(cmv1.NewHypershift().Enabled(true))
		c.Version(version4130)
	})

	// hypershiftClusterReady has no available upgrades
	var hypershiftClusterReady = test.FormatClusterList([]*cmv1.Cluster{mockClusterReady})

	version4130WithUpgrades := version4130.AvailableUpgrades("4.13.1")
	mockClusterReadyWithUpgrades := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateReady)
		c.Hypershift(cmv1.NewHypershift().Enabled(true))
		c.Version(version4130WithUpgrades)
	})

	// hypershiftClusterReadyWithUpdates has one available upgrade
	var hypershiftClusterReadyWithUpdates = test.FormatClusterList([]*cmv1.Cluster{mockClusterReadyWithUpgrades})

	mockClassicCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateReady)
		c.Hypershift(cmv1.NewHypershift().Enabled(false))
	})

	var classicCluster = test.FormatClusterList([]*cmv1.Cluster{mockClassicCluster})

	BeforeEach(func() {
		testRuntime.InitRuntime()
	})

	It("Fails if cluster is not hypershift and we are using hypershift specific flags", func() {
		args.schedule = cronSchedule
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("The '--schedule' option is only supported for Hosted Control Planes"))
	})
	It("Fails if we are using minor version flag for manual upgrades", func() {
		args.schedule = ""
		args.allowMinorVersionUpdates = true
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("The '--allow-minor-version-upgrades' " +
			"option needs to be used with --schedule"))
	})
	It("Fails if we are mixing scheduling type flags", func() {
		args.schedule = cronSchedule
		args.scheduleDate = "31 Jan"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("The '--schedule-date' and '--schedule-time' " +
			"options are mutually exclusive with '--schedule'"))
	})
	It("Fails if we are mixing automatic scheduling and version flags", func() {
		args.schedule = cronSchedule
		args.scheduleDate = ""
		args.version = "4.13.0"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("The '--schedule' " +
			"option is mutually exclusive with '--version'"))
	})
	It("Fails if cluster is not ready", func() {
		args.allowMinorVersionUpdates = false
		args.schedule = ""
		args.scheduleDate = ""
		args.version = ""
		testRuntime.ApiServer.AppendHandlers(
			RespondWithJSON(
				http.StatusOK,
				hypershiftClusterNotReady,
			),
		)
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("Cluster 'cluster1' is not yet ready"))
	})
	It("Cluster is ready but existing upgrade scheduled", func() {
		args.schedule = cronSchedule
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		// An existing policy upgrade
		t, err := time.Parse(time.RFC3339, "2023-06-02T12:30:00Z")
		Expect(err).To(BeNil())
		upgradeState := cmv1.NewUpgradePolicyState().Value("pending")
		cpUpgradePolicy, err := cmv1.NewControlPlaneUpgradePolicy().UpgradeType(cmv1.UpgradeTypeControlPlane).
			NextRun(t).Version("4.12.18").State(upgradeState).ScheduleType(cmv1.ScheduleTypeAutomatic).
			Schedule("30 12 * * *").Build()
		Expect(err).To(BeNil())
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{cpUpgradePolicy})))
		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(BeNil())
		Expect(stdout).To(BeEmpty())
		Expect(stderr).To(ContainSubstring(
			"There is already a pending upgrade to version 4.12.18 on 2023-06-02 12:30 UTC"))
	})
	It("Cluster is ready and with automatic scheduling but bad cron format", func() {
		// not a valid cron
		args.schedule = "* a"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))

		_, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("Schedule '* a' is not a valid cron expression"))
	})

	It("Cluster is ready and with manual scheduling but bad format", func() {
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = "Jan 23"
		args.scheduleTime = timeSchedule
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))

		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		// Missing the upgrade type
		Expect(err.Error()).To(ContainSubstring(
			"schedule date should use the format 'yyyy-mm-dd'\n   Schedule time should use the format 'HH:mm'"))
	})
	It("Cluster is ready and with manual scheduling but no upgrades available", func() {
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = dateSchedule
		args.scheduleTime = timeSchedule
		// The version we want to update to
		args.version = "4.13.4"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(BeNil())
		Expect(stdout).To(BeEmpty())
		Expect(stderr).To(ContainSubstring("There are no available upgrades"))
	})
	It("Cluster is ready and with automatic scheduling, no upgrades available -> still success", func() {
		args.schedule = "20 5 * * *"
		args.scheduleDate = ""
		args.scheduleTime = ""
		args.version = ""
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
		// POST -
		// /api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/control_plane/upgrade_policies?dryRun=true
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))
		// POST - /api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/control_plane/upgrade_policies
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, ""))
		// GET - /api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		// PATCH - /api/clusters_mgmt/v1/clusters
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(BeNil())
		Expect(stdout).To(ContainSubstring("INFO: Upgrade successfully scheduled for cluster 'cluster1'"))
		Expect(stderr).To(BeEmpty())
	})
	It("Cluster is ready and with manual scheduling and available upgrades but a wrong version in input", func() {
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = dateSchedule
		args.scheduleTime = timeSchedule
		// The version we want to update to
		args.version = "4.13.4"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReadyWithUpdates))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))

		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("Expected a valid version to upgrade cluster to.\nValid versions: [4.13.1]"))
	})
	It("Cluster is ready and with manual scheduling and one available upgrade but cluster not found", func() {
		args.schedule = ""
		// Not a valid date format
		args.scheduleDate = dateSchedule
		args.scheduleTime = timeSchedule
		// The version we want to update to
		args.version = "4.13.1"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReadyWithUpdates))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))

		cpUpgradePolicy, err := cmv1.NewControlPlaneUpgradePolicy().UpgradeType(cmv1.UpgradeTypeControlPlane).
			ScheduleType(cmv1.ScheduleTypeAutomatic).Schedule("30 12 * * *").
			EnableMinorVersionUpgrades(false).Build()
		Expect(err).To(BeNil())
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, test.FormatResource(cpUpgradePolicy)))
		// return an empty list to indicate that no cluster is found
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, emptyClusterList))
		err = runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("There is no cluster with identifier or name"))
	})
	It("Fails if node-drain-grace-period flag is specified for hypershift clusters", func() {
		args.schedule = "20 5 * * *"
		args.scheduleDate = ""
		args.scheduleTime = ""
		args.version = ""
		Cmd.Flags().Set("node-drain-grace-period", "45 minutes")
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReadyWithUpdates))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("node-drain-grace-period flag is not supported to hosted clusters"))
	})
})

func formatControlPlaneUpgradePolicyList(upgradePolicies []*cmv1.ControlPlaneUpgradePolicy) string {
	var policiesJson bytes.Buffer

	cmv1.MarshalControlPlaneUpgradePolicyList(upgradePolicies, &policiesJson)

	return fmt.Sprintf(`
	{
		"kind": "ControlPlaneUpgradePolicyList",
		"page": 1,
		"size": %d,
		"total": %d,
		"items": %s
	}`, len(upgradePolicies), len(upgradePolicies), policiesJson.String())
}

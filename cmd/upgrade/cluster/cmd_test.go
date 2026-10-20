// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"
	"github.com/spf13/pflag"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
	upgradecheck "github.com/openshift/rosa/pkg/upgrade"
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

	mockClassicClusterWithUpgrades := test.MockCluster(func(c *cmv1.ClusterBuilder) {
		c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
		c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
		c.State(cmv1.ClusterStateReady)
		c.Hypershift(cmv1.NewHypershift().Enabled(false))
		c.Version(version4130WithUpgrades)
	})
	classicClusterWithUpgrades := test.FormatClusterList([]*cmv1.Cluster{mockClassicClusterWithUpgrades})

	BeforeEach(func() {
		testRuntime.InitRuntime()
		args.dryRun = false
		args.checkPDBs = false
		interactive.SetEnabled(false)
		Cmd.Flags().VisitAll(func(flag *pflag.Flag) { flag.Changed = false })
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			return nil, nil
		}
	})

	It("runs the standalone PDB check without version selection or upgrade requests", func() {
		args.checkPDBs = true
		checkCalled := false
		runPotentialPDBBlockerCheck = func(_ context.Context, apiURL string) ([]upgradecheck.PDBBlocker, error) {
			checkCalled = true
			Expect(apiURL).To(BeEmpty())
			return nil, nil
		}
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(checkCalled).To(BeTrue())
		Expect(stdout).To(ContainSubstring("PDB check for cluster"))
		Expect(stdout).To(ContainSubstring("completed with no current blockers"))
		Expect(stderr).To(BeEmpty())
		requests := testRuntime.ApiServer.ReceivedRequests()
		Expect(requests).To(HaveLen(1))
		Expect(requests[0].URL.Path).To(ContainSubstring("/clusters"))
	})

	It("runs the standalone PDB check even when an upgrade is already scheduled", func() {
		args.checkPDBs = true
		checkCalled := false
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			checkCalled = true
			return nil, nil
		}
		// The only configured response is the cluster lookup. A scheduled-upgrade request would fail this test.
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))

		_, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(checkCalled).To(BeTrue())
		Expect(testRuntime.ApiServer.ReceivedRequests()).To(HaveLen(1))
	})

	It("reports PDB findings as a successful standalone check", func() {
		args.checkPDBs = true
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			return []upgradecheck.PDBBlocker{{
				Namespace: "payments", Name: "api", DisruptionsAllowed: 0,
				Reasons:      []string{"the PDB currently allows zero disruptions"},
				MatchingPods: []string{"payments/api-0"}, Nodes: []string{"node-a"},
			}}, nil
		}
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).NotTo(HaveOccurred())
		Expect(stderr).To(ContainSubstring("oc -n payments get pdb api -o yaml"))
		Expect(stderr).To(ContainSubstring("oc -n payments edit pdb api"))
		Expect(stderr).To(ContainSubstring("Rerun 'rosa upgrade cluster --check-pdbs' after changes."))
		Expect(stdout).To(ContainSubstring("completed with 1 potential blocker(s)"))
		Expect(stdout).NotTo(ContainSubstring("upgrade pre-checks"))
	})

	It("fails an inconclusive standalone PDB check without reporting a clean result", func() {
		args.checkPDBs = true
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			return []upgradecheck.PDBBlocker{{Namespace: "payments", Name: "api"}}, fmt.Errorf("list pods forbidden")
		}
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("PDB check was inconclusive")))
		Expect(stderr).To(ContainSubstring("PDB preflight for cluster"))
		Expect(stderr).To(ContainSubstring("payments/api"))
		Expect(stdout).NotTo(ContainSubstring("no current blockers"))
	})

	It("rejects standalone PDB checking on Hosted Control Plane clusters", func() {
		args.checkPDBs = true
		checkCalled := false
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			checkCalled = true
			return nil, nil
		}
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))

		_, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("supported only for Classic clusters")))
		Expect(checkCalled).To(BeFalse())
	})

	It("rejects upgrade flags with standalone PDB checking before making OCM requests", func() {
		args.checkPDBs = true
		Expect(Cmd.Flags().Set("dry-run", "false")).To(Succeed())

		_, _, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(MatchError(ContainSubstring("cannot be combined with '--dry-run'")))
		Expect(testRuntime.ApiServer.ReceivedRequests()).To(BeEmpty())
	})

	It("Fails if cluster is not hypershift and we are using hypershift specific flags", func() {
		args.schedule = cronSchedule
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicCluster))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(
			ContainSubstring("the '--schedule' option is only supported for Hosted Control Planes"))
	})
	It("Fails if we are using minor version flag for manual upgrades", func() {
		args.schedule = ""
		args.allowMinorVersionUpdates = true
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("the '--allow-minor-version-updates' " +
			"option needs to be used with --schedule"))
	})
	It("Fails if we are mixing scheduling type flags", func() {
		args.schedule = cronSchedule
		args.scheduleDate = "31 Jan"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("the '--schedule-date' and '--schedule-time' " +
			"options are mutually exclusive with '--schedule'"))
	})
	It("Fails if we are mixing automatic scheduling and version flags", func() {
		args.schedule = cronSchedule
		args.scheduleDate = ""
		args.version = "4.13.0"
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReady))
		err := runWithRuntime(testRuntime.RosaRuntime, Cmd)
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("the '--schedule' " +
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
		Expect(err.Error()).To(ContainSubstring("cluster 'cluster1' is not yet ready"))
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
		Expect(stderr).To(Equal("WARN: To check and acknowledge gates prior to scheduling an upgrade, run" +
			" this command with '--dry-run', particularly if you encounter an upgrade failure related to " +
			"acknowledged agreements.\n"))
	})
	It("Cluster is ready and with automatic scheduling, dry run flag doesn't allow cluster upgrade", func() {
		args.schedule = "20 5 * * *"
		args.scheduleDate = ""
		args.scheduleTime = ""
		args.version = ""
		args.dryRun = true
		pdbCheckCalled := false
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			pdbCheckCalled = true
			return nil, fmt.Errorf("PDB check must not run for a hosted control plane upgrade")
		}
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReadyWithUpdates))
		// No existing policy upgrade
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
		// POST -
		// /api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/control_plane/upgrade_policies?dryRun=true
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))
		// POST - /api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8/control_plane/upgrade_policies
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, ""))
		// GET - /api/clusters_mgmt/v1/clusters/24vf9iitg3p6tlml88iml6j6mu095mh8
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReadyWithUpdates))
		// PATCH - /api/clusters_mgmt/v1/clusters
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftClusterReadyWithUpdates))

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(BeNil())
		Expect(stdout).To(ContainSubstring("INFO: Running in dry-run mode. Will not perform cluster upgrade"))
		Expect(stdout).To(ContainSubstring("INFO: ROSA upgrade pre-checks for cluster '24vf9iitg3p6tlml88iml6j6mu095mh8' completed. " +
			"Please wait 1 to 2 minutes, then rerun this command without the '--dry-run' flag, to allow time for the " +
			"acknowledged agreements to be reflected.\n"))
		Expect(stdout).To(ContainSubstring("PDB drain check is not applicable to hosted control plane upgrades"))
		Expect(pdbCheckCalled).To(BeFalse())
		Expect(stderr).To(BeEmpty())
	})
	It("Warns and completes Classic dry-run for a PDB at its limit", func() {
		args.schedule = ""
		args.scheduleDate = ""
		args.scheduleTime = ""
		args.version = "4.13.1"
		args.dryRun = true
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			return []upgradecheck.PDBBlocker{{
				Namespace: "payments", Name: "api", ExpectedPods: 2, CurrentHealthy: 2, DesiredHealthy: 2,
				Budget: "minAvailable=100%", Reasons: []string{"the PDB currently allows zero disruptions"},
			}}, nil
		}
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterWithUpgrades))
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"kind":"UpgradePolicyList","page":1,"size":0,"total":0,"items":[]}`))
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(ContainSubstring("oc -n payments get pdb api -o yaml"))
		Expect(stderr).To(ContainSubstring("oc -n payments edit pdb api"))
		Expect(stderr).To(ContainSubstring("A PDB blocker may delay node draining"))
		Expect(stderr).To(ContainSubstring("may be forcibly evicted"))
		Expect(stdout).To(ContainSubstring("completed with potential PDB blockers"))
		Expect(stdout).NotTo(ContainSubstring("rerun this command without the '--dry-run' flag"))
	})
	It("Completes Classic dry-run with an inconclusive PDB warning when the scan cannot run", func() {
		args.schedule = ""
		args.scheduleDate = ""
		args.scheduleTime = ""
		args.version = "4.13.1"
		args.dryRun = true
		runPotentialPDBBlockerCheck = func(context.Context, string) ([]upgradecheck.PDBBlocker, error) {
			return nil, fmt.Errorf("Kubernetes API access is unavailable")
		}
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterWithUpgrades))
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
			`{"kind":"UpgradePolicyList","page":1,"size":0,"total":0,"items":[]}`))
		testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))

		stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
		Expect(err).To(BeNil())
		Expect(stderr).To(ContainSubstring("PDB preflight for cluster"))
		Expect(stdout).To(ContainSubstring("PDB check was inconclusive"))
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

	Context("HCP managed policies IAM upgrade gating", func() {
		const operatorCredRequests = `{
			"kind":"STSCredentialRequestList","page":1,"size":1,"total":1,
			"items":[{
				"kind":"STSCredentialRequest",
				"name":"operator-1",
				"operator":{"name":"cloud-credentials","namespace":"openshift-ingress-operator"}
			}]
		}`
		var hypershiftManagedSTSCluster string
		var mockAWS *mock.MockClient

		BeforeEach(func() {
			interactive.SetModeKey("")
			Expect(Cmd.Flags().Set("mode", "")).To(Succeed())
			Cmd.Flag("mode").Changed = false
			Expect(Cmd.Flags().Set("node-drain-grace-period", "")).To(Succeed())
			Cmd.Flag("node-drain-grace-period").Changed = false

			mockAWS = testRuntime.RosaRuntime.AWSClient.(*mock.MockClient)

			managedCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
				c.AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
					RoleARN("arn:aws:iam::123456789012:role/ManagedOpenShift-HCP-ROSA-Installer-Role").
					ManagedPolicies(true)))
				c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
				c.State(cmv1.ClusterStateReady)
				c.Hypershift(cmv1.NewHypershift().Enabled(true))
				c.Version(version4130)
			})
			hypershiftManagedSTSCluster = test.FormatClusterList([]*cmv1.Cluster{managedCluster})
		})

		AfterEach(func() {
			interactive.SetModeKey("")
			Expect(Cmd.Flags().Set("mode", "")).To(Succeed())
			Cmd.Flag("mode").Changed = false
			args.schedule = ""
			args.scheduleDate = ""
			args.scheduleTime = ""
			args.version = ""
			args.dryRun = false
		})

		appendManagedPolicyValidationHandlers := func() {
			policy, err := cmv1.NewAWSSTSPolicy().ID("123").Type("").Build()
			Expect(err).NotTo(HaveOccurred())
			opPolicy, err := cmv1.NewAWSSTSPolicy().ID("123").Type("OperatorRole").Build()
			Expect(err).NotTo(HaveOccurred())
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatAWSSTSPolicyList([]*cmv1.AWSSTSPolicy{policy})))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, operatorCredRequests))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatAWSSTSPolicyList([]*cmv1.AWSSTSPolicy{opPolicy})))
			mockAWS.EXPECT().ValidateHCPAccountRolesManagedPolicies("ManagedOpenShift", gomock.Any()).Return(nil)
			mockAWS.EXPECT().ValidateOperatorRolesManagedPolicies(
				gomock.Any(), gomock.Any(), gomock.Any(), true,
			).Return(nil)
		}

		It("warns and discards --mode for HCP managed policies", func() {
			args.schedule = "20 5 * * *"
			args.scheduleDate = ""
			args.scheduleTime = ""
			args.version = ""
			interactive.SetModeKey(interactive.ModeAuto)
			Expect(Cmd.Flags().Set("mode", interactive.ModeAuto)).To(Succeed())

			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
			appendManagedPolicyValidationHandlers()
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, ""))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))

			stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("INFO: Upgrade successfully scheduled for cluster 'cluster1'"))
			Expect(stderr).To(ContainSubstring(
				"The '--mode' flag is not applicable for Hosted Control Plane clusters " +
					"with managed policies and will be ignored"))
			Expect(stdout).NotTo(ContainSubstring("compatible with upgrade"))
			Expect(stdout).NotTo(ContainSubstring("An upgrade isn't needed"))
		})

		It("warns and continues when an unsupported --mode is set for HCP managed policies", func() {
			args.schedule = "20 5 * * *"
			args.scheduleDate = ""
			args.scheduleTime = ""
			args.version = ""
			interactive.SetModeKey("unsupported-mode")
			Expect(Cmd.Flags().Set("mode", "unsupported-mode")).To(Succeed())

			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
			appendManagedPolicyValidationHandlers()
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, ""))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))

			stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("INFO: Upgrade successfully scheduled for cluster 'cluster1'"))
			Expect(stderr).To(ContainSubstring(
				"The '--mode' flag is not applicable for Hosted Control Plane clusters " +
					"with managed policies and will be ignored"))
		})

		It("skips IAM mode prompt noise when --mode is not set for HCP managed policies", func() {
			args.schedule = "20 5 * * *"
			args.scheduleDate = ""
			args.scheduleTime = ""
			args.version = ""

			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))
			appendManagedPolicyValidationHandlers()
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNoContent, ""))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusCreated, ""))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))

			stdout, stderr, err := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(err).To(BeNil())
			Expect(stdout).To(ContainSubstring("INFO: Upgrade successfully scheduled for cluster 'cluster1'"))
			Expect(stderr).NotTo(ContainSubstring("The '--mode' flag is not applicable"))
			Expect(stdout).NotTo(ContainSubstring("compatible with upgrade"))
			Expect(stdout).NotTo(ContainSubstring("An upgrade isn't needed"))
			Expect(stdout).NotTo(ContainSubstring("have attached managed policies"))
		})

		It("fails when managed policy attachments are missing", func() {
			args.schedule = "20 5 * * *"
			args.scheduleDate = ""
			args.scheduleTime = ""
			args.version = ""

			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, hypershiftManagedSTSCluster))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				formatControlPlaneUpgradePolicyList([]*cmv1.ControlPlaneUpgradePolicy{})))

			policy, err := cmv1.NewAWSSTSPolicy().ID("123").Type("").Build()
			Expect(err).NotTo(HaveOccurred())
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK,
				test.FormatAWSSTSPolicyList([]*cmv1.AWSSTSPolicy{policy})))
			mockAWS.EXPECT().ValidateHCPAccountRolesManagedPolicies("ManagedOpenShift", gomock.Any()).
				Return(fmt.Errorf("role 'x' is missing the attached managed policy 'y'"))

			_, _, runErr := test.RunWithOutputCapture(runWithRuntime, testRuntime.RosaRuntime, Cmd)
			Expect(runErr).ToNot(BeNil())
			Expect(runErr.Error()).To(ContainSubstring("failed while validating managed policies"))
			Expect(runErr.Error()).To(ContainSubstring("missing the attached managed policy"))
		})
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

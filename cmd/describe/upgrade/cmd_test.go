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
			upgradeState := cmv1.NewUpgradePolicyState().Value("scheduled")
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").
				State(upgradeState).NextRun(nowUTC).Build()
			Expect(err).To(BeNil())
			result := formatHypershiftUpgrade(npUpgradePolicy)
			Expect(result).To(Equal(
				fmt.Sprintf(
					`                ID:                                                                 id1
		Cluster ID:                        
		Schedule Type:                     
		Next Run:                          %s
		Upgrade State:                     scheduled

                Version:                           4.12.19
`, nowUTC.Format("2006-01-02 15:04 MST"))))
		})
		It("Node pool upgrade is scheduled with a date", func() {
			format.TruncatedDiff = false
			nowUTC := time.Now().UTC()
			upgradeState := cmv1.NewUpgradePolicyState().Value("scheduled")
			npUpgradePolicy, err := cmv1.NewNodePoolUpgradePolicy().ID("id1").Version("4.12.19").
				State(upgradeState).NextRun(nowUTC).Schedule(nowUTC.Format("2006-01-02 15:04 MST")).
				EnableMinorVersionUpgrades(true).Build()
			Expect(err).To(BeNil())
			result := formatHypershiftUpgrade(npUpgradePolicy)
			Expect(result).To(Equal(
				fmt.Sprintf(
					`                ID:                                                                 id1
		Cluster ID:                        
		Schedule Type:                     
		Next Run:                          %s
		Upgrade State:                     scheduled
		Schedule At:                       %s
        Enable minor version upgrades:     true

                Version:                           4.12.19
`, nowUTC.Format("2006-01-02 15:04 MST"), nowUTC.Format("2006-01-02 15:04 MST"))))
		})
	})
	Context("Describe Hypershift upgrade", func() {
		var testRuntime test.TestingRuntime
		var clusterID = "cluster1"
		var nodePoolID = "nodepool85"

		const nodePoolResponse = `{
						  "kind": "NodePool",
						  "href": "/api/clusters_mgmt/v1/clusters/243nmgjr5v2q9rn5sf3456euj2lcq5tn/node_pools/workers",
						  "id": "workers",
						  "replicas": 2,
						  "auto_repair": true,
						  "aws_node_pool": {
							"instance_type": "m5.xlarge",
							"instance_profile": "rosa-service-managed-integration-243nmgjr5v2q9rn5sf3456euj2lcq5tn-ad-int1-worker",
							"tags": {
							  "api.openshift.com/environment": "integration",
							  "api.openshift.com/id": "243nmgjr5v2q9rn5sf3456euj2lcq5tn",
							  "api.openshift.com/legal-entity-id": "1jIHnIbrnLH9kQD57W0BuPm78f1",
							  "api.openshift.com/name": "ad-int1",
							  "api.openshift.com/nodepool-hypershift": "ad-int1-workers",
							  "api.openshift.com/nodepool-ocm": "workers",
							  "red-hat-clustertype": "rosa",
							  "red-hat-managed": "true"
							}
						  },
						  "availability_zone": "us-west-2a",
						  "subnet": "subnet-0e3a4046c1c2f1078",
						  "status": {
							"current_replicas": 0,
							"message": "WaitingForAvailableMachines: NodeProvisioning"
						  },
						  "version": {
							"kind": "VersionLink",
							"id": "openshift-v4.12.%s",
							"href": "/api/clusters_mgmt/v1/versions/openshift-v4.12.%s"
						  },
						  "tuning_configs": []
						}`

		// nolint:lll
		const nodePoolUpgradePolicy = `{
  "kind": "NodePoolUpgradePolicyList",
  "page": 1,
  "size": 1,
  "total": 1,
  "items": [
    {
      "kind": "NodePoolUpgradePolicy",
      "id": "e2800d05-3534-11ee-b9bc-0a580a811709",
      "href": "/api/clusters_mgmt/v1/clusters/25f96obptkqc5mh9vdc779jiqb3sihnn/node_pools/workers/upgrade_policies/e2800d05-3534-11ee-b9bc-0a580a811709",
      "schedule_type": "manual",
      "upgrade_type": "NodePool",
      "version": "4.12.25",
      "next_run": "2023-08-07T15:22:00Z",
      "cluster_id": "25f96obptkqc5mh9vdc779jiqb3sihnn",
      "node_pool_id": "workers",
      "enable_minor_version_upgrades": true,
      "creation_timestamp": "2023-08-07T15:12:54.967835Z",
      "last_update_timestamp": "2023-08-07T15:12:54.967835Z",
      "state": {
        "value": "scheduled",
        "description": "Upgrade scheduled."
      }
    }
  ]
}`

		BeforeEach(func() {
			testRuntime.InitRuntime()
		})
		It("Upgrade policy found, no error", func() {
			args.nodePool = nodePoolID
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolResponse))
			testRuntime.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, nodePoolUpgradePolicy))
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

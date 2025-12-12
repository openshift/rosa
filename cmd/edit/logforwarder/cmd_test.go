package logforwarder

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Edit Log Forwarder", func() {
	Context("EditLogForwarderRunner", func() {

		clusterId := "test-cluster"
		logForwarderId := "test-log-forwarder"

		mockClusterReady := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.AWS(cmv1.NewAWS().SubnetIDs("subnet-0b761d44d3d9a4663", "subnet-0f87f640e56934cbc"))
			c.Region(cmv1.NewCloudRegion().ID("us-east-1"))
			c.State(cmv1.ClusterStateReady).ID(clusterId)
		})

		var t *test.TestingRuntime

		BeforeEach(func() {
			t = test.NewTestRuntime()
		})

		AfterEach(func() {
			t.SetCluster("", nil)
		})

		It("Should fail when no log forwarder ID is provided", func() {
			t.SetCluster(clusterId, mockClusterReady)

			cmd := NewEditLogForwarderCommand()
			Expect(cmd.Flag("log-fwd-config").Value.Set("testdata/valid-config.yaml")).To(Succeed())

			err := EditLogForwarderRunner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected exactly one argument"))
		})

		It("Should fail when multiple log forwarder IDs are provided", func() {
			t.SetCluster(clusterId, mockClusterReady)

			cmd := NewEditLogForwarderCommand()
			Expect(cmd.Flag("log-fwd-config").Value.Set("testdata/valid-config.yaml")).To(Succeed())

			err := EditLogForwarderRunner(context.Background(), t.RosaRuntime, cmd, []string{logForwarderId, "extra-id"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected exactly one argument"))
		})
	})
})

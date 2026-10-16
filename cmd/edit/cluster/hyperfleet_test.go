package cluster

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
)

var _ = Describe("hyperfleet dispatch", func() {
	var (
		origEnabled     func() bool
		origEditCluster func(*cobra.Command)
	)

	BeforeEach(func() {
		origEnabled = hyperfleetEnabled
		origEditCluster = hfEditCluster
	})

	AfterEach(func() {
		hyperfleetEnabled = origEnabled
		hfEditCluster = origEditCluster
	})

	It("routes to hfEditCluster when hyperfleet is enabled", func() {
		called := false
		hyperfleetEnabled = func() bool { return true }
		hfEditCluster = func(_ *cobra.Command) { called = true }

		dispatch(makeCmd(), nil)

		Expect(called).To(BeTrue())
	})
})

var _ = Describe("hyperfleet cluster update spot termination queue", func() {
	makeUpdateCommand := func() *cobra.Command {
		cmd := &cobra.Command{}
		cmd.Flags().String("spot-termination-queue-url", "", "")
		return cmd
	}

	It("accepts the queue URL and builds the platform patch", func() {
		cmd := makeUpdateCommand()
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789012/rosa-spot-queue"
		Expect(cmd.Flags().Set("spot-termination-queue-url", queueURL)).To(Succeed())

		handler := &hyperfleetClusterUpdate{cmd: cmd}
		input := &hfpathbind.ClusterUpdateInput{}

		Expect(handler.PreRequest(context.Background(), nil, input)).To(Succeed())
		Expect(input.TerminationHandlerQueueURL).To(Equal(queueURL))

		patch, err := handler.buildSpecPatch(input)
		Expect(err).NotTo(HaveOccurred())

		var body map[string]any
		Expect(json.Unmarshal(patch, &body)).To(Succeed())
		spec := body["spec"].(map[string]any)
		hostedCluster := spec["hostedCluster"].(map[string]any)
		platform := hostedCluster["platform"].(map[string]any)
		aws := platform["aws"].(map[string]any)
		Expect(aws["terminationHandlerQueueURL"]).To(Equal(queueURL))
	})

	It("allows clearing the queue URL", func() {
		cmd := makeUpdateCommand()
		Expect(cmd.Flags().Set("spot-termination-queue-url", "")).To(Succeed())

		handler := &hyperfleetClusterUpdate{cmd: cmd}
		input := &hfpathbind.ClusterUpdateInput{}

		Expect(handler.PreRequest(context.Background(), nil, input)).To(Succeed())
		Expect(input.TerminationHandlerQueueURL).To(BeEmpty())

		patch, err := handler.buildSpecPatch(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(patch)).To(ContainSubstring(`"terminationHandlerQueueURL":""`))
	})

	It("rejects an invalid queue URL", func() {
		cmd := makeUpdateCommand()
		Expect(cmd.Flags().Set(
			"spot-termination-queue-url",
			"http://sqs.us-east-1.amazonaws.com/123456789012/rosa-spot-queue",
		)).To(Succeed())

		handler := &hyperfleetClusterUpdate{cmd: cmd}

		err := handler.PreRequest(context.Background(), nil, &hfpathbind.ClusterUpdateInput{})

		Expect(err).To(MatchError(ContainSubstring("invalid value for '--spot-termination-queue-url'")))
	})
})

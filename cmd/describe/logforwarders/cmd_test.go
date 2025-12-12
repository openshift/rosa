package logforwarders

import (
	"bytes"
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
	. "github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/test"
)

var _ = Describe("Describe log forwarding", func() {
	const (
		s3LogForwarderOutput = `
S3 Bucket Prefix:                     /rosa/log-forwarding
S3 Bucket Name:                      my-log-bucket
Applications:                        audit infrastructure
Groups:                              (applications,v0)
`
		cloudwatchLogForwarderOutput = `
Cloudwatch Log Group Name:           my-cloudwatch-log-group
Cloudwatch Log Distribution Role Arn: arn:aws:iam::123456789012:role/cloudwatch-log-role
Applications:                        audit infrastructure
Groups:                              (applications,v0)
`
	)
	Context("describe", func() {
		format.TruncatedDiff = false

		mockReadyCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.ID("123")
			c.Region(cmv1.NewCloudRegion().ID(aws.DefaultRegion))
			c.State(cmv1.ClusterStateReady)
		})
		classicClusterReady := test.FormatClusterList([]*cmv1.Cluster{mockReadyCluster})
		mockNotReadyCluster := test.MockCluster(func(c *cmv1.ClusterBuilder) {
			c.ID("123")
			c.Region(cmv1.NewCloudRegion().ID(aws.DefaultRegion))
			c.State(cmv1.ClusterStateInstalling)
		})
		classicClusterNotReady := test.FormatClusterList([]*cmv1.Cluster{mockNotReadyCluster})

		s3LogForwarder, err := cmv1.NewLogForwarder().
			S3(cmv1.NewLogForwarderS3Config().
				BucketName("my-log-bucket").
				BucketPrefix("/rosa/log-forwarding")).
			Applications("audit", "infrastructure").
			Groups(cmv1.NewLogForwarderGroup().
				ID("applications").
				Version("0")).Build()
		Expect(err).To(BeNil())
		s3LogForwarderResponse := test.FormatLogForwarder(s3LogForwarder)

		cloudwatchLogForwarder, err := cmv1.NewLogForwarder().
			Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
				LogGroupName("my-cloudwatch-log-group").
				LogDistributionRoleArn("arn:aws:iam::123456789012:role/cloudwatch-log-role")).
			Applications("audit", "infrastructure").
			Groups(cmv1.NewLogForwarderGroup().
				ID("applications").
				Version("0")).Build()
		Expect(err).To(BeNil())
		cloudwatchLogForwarderResponse := test.FormatLogForwarder(cloudwatchLogForwarder)

		var t *test.TestingRuntime
		BeforeEach(func() {
			t = test.NewTestRuntime()
			SetOutput("")
		})

		It("Fails if log forwarder ID has not been specified", func() {
			runner := DescribeLogForwarderRunner(NewDescribeLogForwarderUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			err = runner(context.Background(), t.RosaRuntime, NewDescribeLogForwarderCommand(), []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("you need to specify a log forwarder ID"))
		})

		It("Fails if log forwarder ID is invalid", func() {
			runner := DescribeLogForwarderRunner(NewDescribeLogForwarderUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeLogForwarderCommand()
			cmd.Flag("log-forwarder").Value.Set("INVALID_ID123")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(BeNil())
			Expect(
				err.Error(),
			).To(Equal(
				"log forwarder identifier 'INVALID_ID123' isn't valid: it must contain only lowercase letters and digits",
			))
		})

		It("Cluster not ready", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterNotReady))
			runner := DescribeLogForwarderRunner(NewDescribeLogForwarderUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeLogForwarderCommand()
			cmd.Flag("log-forwarder").Value.Set("2n4b8f8ai80cs6kmjmdgqlqplh73r411")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal("cluster '123' is not yet ready"))
		})

		It("Log forwarder not found", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusNotFound, ""))
			runner := DescribeLogForwarderRunner(NewDescribeLogForwarderUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeLogForwarderCommand()
			cmd.Flag("log-forwarder").Value.Set("2n4b8f8ai80cs6kmjmdgqlqplh73r411")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to get log forwarder"))
		})

		It("S3 Log forwarder found", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, s3LogForwarderResponse))
			runner := DescribeLogForwarderRunner(NewDescribeLogForwarderUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeLogForwarderCommand()
			cmd.Flag("log-forwarder").Value.Set("2n4b8f8ai80cs6kmjmdgqlqplh73r411")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal(s3LogForwarderOutput))
		})

		It("CloudWatch Log forwarder found", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, cloudwatchLogForwarderResponse))
			runner := DescribeLogForwarderRunner(NewDescribeLogForwarderUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeLogForwarderCommand()
			cmd.Flag("log-forwarder").Value.Set("2n4b8f8ai80cs6kmjmdgqlqplh73r411")
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal(cloudwatchLogForwarderOutput))
		})

		It("Log forwarder found through argv", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, s3LogForwarderResponse))
			args := NewDescribeLogForwarderUserOptions()
			runner := DescribeLogForwarderRunner(args)
			cmd := NewDescribeLogForwarderCommand()
			cmd.Flag("cluster").Value.Set("123")
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			err = runner(context.Background(), t.RosaRuntime, cmd,
				[]string{
					"2n4b8f8ai80cs6kmjmdgqlqplh73r411",
				})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(stdout).To(Equal(s3LogForwarderOutput))
		})

		It("Log forwarder found json output", func() {
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, classicClusterReady))
			t.ApiServer.AppendHandlers(RespondWithJSON(http.StatusOK, s3LogForwarderResponse))
			runner := DescribeLogForwarderRunner(NewDescribeLogForwarderUserOptions())
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			cmd := NewDescribeLogForwarderCommand()
			cmd.Flag("cluster").Value.Set(mockReadyCluster.ID())
			cmd.Flag("output").Value.Set("json")
			err = runner(context.Background(), t.RosaRuntime, cmd, []string{"2n4b8f8ai80cs6kmjmdgqlqplh73r411"})
			Expect(err).ToNot(HaveOccurred())
			stdout, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			var logForwarderJson bytes.Buffer
			cmv1.MarshalLogForwarder(s3LogForwarder, &logForwarderJson)
			Expect(stdout).To(Equal(logForwarderJson.String() + "\n"))
		})
	})
})

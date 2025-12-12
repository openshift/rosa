package logforwarding

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("LogForwarder Output", func() {
	Context("LogForwarderObjectAsString", func() {
		It("Returns formatted S3 log forwarder output", func() {
			s3LogForwarder, err := cmv1.NewLogForwarder().
				S3(cmv1.NewLogForwarderS3Config().
					BucketName("my-log-bucket").
					BucketPrefix("/rosa/log-forwarding")).
				Applications("audit", "infrastructure").
				Groups(cmv1.NewLogForwarderGroup().ID("applications").Version("0"),
					cmv1.NewLogForwarderGroup().ID("api").Version("1"),
				).Build()
			Expect(err).To(BeNil())

			result := LogForwarderObjectAsString(s3LogForwarder)

			expected := "\n" +
				"S3 Bucket Prefix:                     /rosa/log-forwarding\n" +
				"S3 Bucket Name:                      my-log-bucket\n" +
				"Applications:                        audit infrastructure\n" +
				"Groups:                              (applications,v0) (api,v1)\n"

			Expect(result).To(Equal(expected))
		})

		It("Returns formatted CloudWatch log forwarder output", func() {
			cloudwatchLogForwarder, err := cmv1.NewLogForwarder().
				Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
					LogGroupName("my-cloudwatch-log-group").
					LogDistributionRoleArn("arn:aws:iam::123456789012:role/cloudwatch-log-role")).
				Applications("audit", "infrastructure").
				Groups(cmv1.NewLogForwarderGroup().
					ID("applications").
					Version("0")).Build()
			Expect(err).To(BeNil())

			result := LogForwarderObjectAsString(cloudwatchLogForwarder)

			expected := "\n" +
				"Cloudwatch Log Group Name:           my-cloudwatch-log-group\n" +
				"Cloudwatch Log Distribution Role Arn: arn:aws:iam::123456789012:role/cloudwatch-log-role\n" +
				"Applications:                        audit infrastructure\n" +
				"Groups:                              (applications,v0)\n"

			Expect(result).To(Equal(expected))
		})

		It("Returns S3 output with empty bucket prefix", func() {
			s3LogForwarderNoPrefix, err := cmv1.NewLogForwarder().
				S3(cmv1.NewLogForwarderS3Config().
					BucketName("my-log-bucket")).
				Applications("audit").Build()
			Expect(err).To(BeNil())

			result := LogForwarderObjectAsString(s3LogForwarderNoPrefix)

			expected := "\n" +
				"S3 Bucket Prefix:                     \n" +
				"S3 Bucket Name:                      my-log-bucket\n" +
				"Applications:                        audit\n"

			Expect(result).To(Equal(expected))
		})

		It("Returns formatted S3 log forwarder output with status", func() {
			s3LogForwarder, err := cmv1.NewLogForwarder().
				S3(cmv1.NewLogForwarderS3Config().
					BucketName("my-log-bucket").
					BucketPrefix("/rosa/log-forwarding")).
				Applications("audit", "infrastructure").
				Status(cmv1.NewLogForwarderStatus().
					Message("Log forwarder is active").
					ResolvedApplications("audit", "infrastructure", "application")).
				Build()
			Expect(err).To(BeNil())

			result := LogForwarderObjectAsString(s3LogForwarder)

			expected := "\n" +
				"S3 Bucket Prefix:                     /rosa/log-forwarding\n" +
				"S3 Bucket Name:                      my-log-bucket\n" +
				"Applications:                        audit infrastructure\n" +
				"Status Message:                      Log forwarder is active\n" +
				"Resolved Applications:               audit infrastructure application\n"

			Expect(result).To(Equal(expected))
		})

		It("Returns formatted CloudWatch log forwarder output with status", func() {
			cloudwatchLogForwarder, err := cmv1.NewLogForwarder().
				Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
					LogGroupName("my-cloudwatch-log-group").
					LogDistributionRoleArn("arn:aws:iam::123456789012:role/cloudwatch-log-role")).
				Applications("audit").
				Status(cmv1.NewLogForwarderStatus().
					Message("CloudWatch forwarding enabled").
					ResolvedApplications("audit", "application")).
				Build()
			Expect(err).To(BeNil())

			result := LogForwarderObjectAsString(cloudwatchLogForwarder)

			expected := "\n" +
				"Cloudwatch Log Group Name:           my-cloudwatch-log-group\n" +
				"Cloudwatch Log Distribution Role Arn: arn:aws:iam::123456789012:role/cloudwatch-log-role\n" +
				"Applications:                        audit\n" +
				"Status Message:                      CloudWatch forwarding enabled\n" +
				"Resolved Applications:               audit application\n"

			Expect(result).To(Equal(expected))
		})

		It("Returns output with only status message when resolved applications is empty", func() {
			logForwarder, err := cmv1.NewLogForwarder().
				S3(cmv1.NewLogForwarderS3Config().
					BucketName("test-bucket")).
				Status(cmv1.NewLogForwarderStatus().
					Message("Partially configured")).
				Build()
			Expect(err).To(BeNil())

			result := LogForwarderObjectAsString(logForwarder)

			expected := "\n" +
				"S3 Bucket Prefix:                     \n" +
				"S3 Bucket Name:                      test-bucket\n" +
				"Status Message:                      Partially configured\n"

			Expect(result).To(Equal(expected))
		})
	})
})

/*
Copyright (c) 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/logforwarding"
)

var _ = Describe("BuildLogForwarder", func() {
	Context("When input is nil", func() {
		It("Should return empty builder", func() {
			builder := BuildLogForwarder(nil)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())
			Expect(out.Applications()).To(BeEmpty())
			_, hasCloudWatch := out.GetCloudwatch()
			Expect(hasCloudWatch).To(BeFalse())
			groups, hasGroups := out.GetGroups()
			Expect(hasGroups).To(BeFalse())
			Expect(groups).To(BeEmpty())
			_, hasS3 := out.GetS3()
			Expect(hasS3).To(BeFalse())
		})
	})

	Context("When input has full configuration", func() {
		It("Should populate all fields correctly", func() {
			input, err := cmv1.NewLogForwarder().
				Applications("app1", "app2").
				Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
					LogGroupName("cw-group").
					LogDistributionRoleArn("cw-arn")).
				Groups(
					cmv1.NewLogForwarderGroup().Version("v1"),
					cmv1.NewLogForwarderGroup().Version("v2"),
				).
				S3(cmv1.NewLogForwarderS3Config().
					BucketName("my-bucket").
					BucketPrefix("logs/")).
				Build()
			Expect(err).ToNot(HaveOccurred())

			builder := BuildLogForwarder(input)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(out.Applications()).To(Equal([]string{"app1", "app2"}))

			cw, hasCloudWatch := out.GetCloudwatch()
			Expect(hasCloudWatch).To(BeTrue())
			Expect(cw.LogGroupName()).To(Equal("cw-group"))
			Expect(cw.LogDistributionRoleArn()).To(Equal("cw-arn"))

			groups, hasGroups := out.GetGroups()
			Expect(hasGroups).To(BeTrue())
			Expect(groups).To(HaveLen(2))
			Expect(groups[0].Version()).To(Equal("v1"))
			Expect(groups[1].Version()).To(Equal("v2"))

			s3, hasS3 := out.GetS3()
			Expect(hasS3).To(BeTrue())
			Expect(s3.BucketName()).To(Equal("my-bucket"))
			Expect(s3.BucketPrefix()).To(Equal("logs/"))
		})
	})

	Context("When input has partial CloudWatch config", func() {
		It("Should handle partial CloudWatch configuration", func() {
			input, err := cmv1.NewLogForwarder().
				Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
					LogGroupName("test-group")).
				Build()
			Expect(err).ToNot(HaveOccurred())

			builder := BuildLogForwarder(input)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			cw, hasCloudWatch := out.GetCloudwatch()
			Expect(hasCloudWatch).To(BeTrue())
			Expect(cw.LogGroupName()).To(Equal("test-group"))
		})
	})

	Context("When input has partial S3 config", func() {
		It("Should handle partial S3 configuration", func() {
			input, err := cmv1.NewLogForwarder().
				S3(cmv1.NewLogForwarderS3Config().
					BucketName("test-bucket")).
				Build()
			Expect(err).ToNot(HaveOccurred())

			builder := BuildLogForwarder(input)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			s3, hasS3 := out.GetS3()
			Expect(hasS3).To(BeTrue())
			Expect(s3.BucketName()).To(Equal("test-bucket"))
		})
	})

	Context("When input has empty applications", func() {
		It("Should handle empty applications list", func() {
			input, err := cmv1.NewLogForwarder().
				Applications().
				Build()
			Expect(err).ToNot(HaveOccurred())

			builder := BuildLogForwarder(input)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(out.Applications()).To(BeEmpty())
		})
	})
})

var _ = Describe("BuildLogForwarder Function Tests", func() {
	Context("Edge cases and error handling", func() {
		It("Should handle empty CloudWatch config properly", func() {
			input, err := cmv1.NewLogForwarder().
				Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig()).
				Build()
			Expect(err).ToNot(HaveOccurred())

			builder := BuildLogForwarder(input)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			_, hasCloudWatch := out.GetCloudwatch()
			Expect(hasCloudWatch).To(BeFalse())
		})

		It("Should handle empty S3 config properly", func() {
			input, err := cmv1.NewLogForwarder().
				S3(cmv1.NewLogForwarderS3Config()).
				Build()
			Expect(err).ToNot(HaveOccurred())

			builder := BuildLogForwarder(input)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			_, hasS3 := out.GetS3()
			Expect(hasS3).To(BeFalse())
		})

		It("Should handle mixed configuration", func() {
			input, err := cmv1.NewLogForwarder().
				Applications("app1").
				Cloudwatch(cmv1.NewLogForwarderCloudWatchConfig().
					LogGroupName("test-group")).
				Groups(cmv1.NewLogForwarderGroup().Version("v1")).
				Build()
			Expect(err).ToNot(HaveOccurred())

			builder := BuildLogForwarder(input)
			out, err := builder.Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(out.Applications()).To(Equal([]string{"app1"}))

			cw, hasCloudWatch := out.GetCloudwatch()
			Expect(hasCloudWatch).To(BeTrue())
			Expect(cw.LogGroupName()).To(Equal("test-group"))

			groups, hasGroups := out.GetGroups()
			Expect(hasGroups).To(BeTrue())
			Expect(groups).To(HaveLen(1))
			Expect(groups[0].Version()).To(Equal("v1"))

			_, hasS3 := out.GetS3()
			Expect(hasS3).To(BeFalse())
		})
	})
})

var _ = Describe("EditLogForwarder", func() {
	Context("EditLogForwarder function", func() {
		It("Should handle empty config", func() {
			client := Client{}
			emptyConfig := logforwarding.LogForwarderYaml{}
			err := client.EditLogForwarder("cluster-123", "log-fwd-123", emptyConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"log forwarding config provided contained no valid log forwarders"))
		})
	})
})

package logforwarding

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var testConfig = `
cloudwatch:
  cloudwatch_log_role_arn: "arn"
  cloudwatch_log_group_name: "abcd"
  applications: ["test3", "test4"]
  groups: ["group-name"]
s3:
  s3_config_bucket_name: "foo"
  s3_config_bucket_prefix: "bar"
  applications: ["test3", "test4"]
  groups: ["group-name2"]
`

func generateLogForwarderGroupVersions() []*cmv1.LogForwarderGroupVersions {
	test1, err := cmv1.NewLogForwarderGroupVersions().Name("Test1").Versions(
		cmv1.NewLogForwarderGroupVersion().ID("v1").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v2").Applications("4", "5", "6"),
		cmv1.NewLogForwarderGroupVersion().ID("v3").Applications("1", "2", "3")).Build()
	Expect(err).NotTo(HaveOccurred())
	test2, err := cmv1.NewLogForwarderGroupVersions().Name("Test2").Versions(
		cmv1.NewLogForwarderGroupVersion().ID("v1").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v2").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v3").Applications("4", "5", "6")).Build()
	Expect(err).NotTo(HaveOccurred())
	test3, err := cmv1.NewLogForwarderGroupVersions().Name("Test3").Versions(
		cmv1.NewLogForwarderGroupVersion().ID("v1").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v2").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v3").Applications("7", "8", "9")).Build()
	Expect(err).NotTo(HaveOccurred())
	return []*cmv1.LogForwarderGroupVersions{
		test1,
		test2,
		test3,
	}
}

var _ = Describe("LogForwarding Config", func() {

	Context("ConstructPodGroupsHelpMessage", func() {
		It("OK: Prints what is expected from a 3 entry long map", func() {
			Expect(ConstructPodGroupsHelpMessage(generateLogForwarderGroupVersions())).To(
				Equal("Test1: 1,2,3\nTest2: 4,5,6\nTest3: 7,8,9\n"),
			)
		})
	})

	Context("ConstructPodGroupsInteractiveOptions", func() {
		It("OK: Parses correctly", func() {
			Expect(ConstructPodGroupsInteractiveOptions(generateLogForwarderGroupVersions())).To(
				Equal([]string{"Test1", "Test2", "Test3"}),
			)
		})
	})

	Context("UnmarshalLogForwarderConfigYaml", func() {
		var tmpDir string

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "-*")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(tmpDir)).ToNot(HaveOccurred())
		})

		// BindCloudWatchLogForwarder and BindS3LogForwarder are also tested here
		It("OK: Unmarshals correctly", func() {
			yamlPath := filepath.Join(tmpDir, "template")
			err := os.WriteFile(yamlPath, []byte(testConfig), 0644)
			Expect(err).ToNot(HaveOccurred())

			config, err := UnmarshalLogForwarderConfigYaml(yamlPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(config).ToNot(BeNil())
			Expect(config.S3).ToNot(BeNil())
			Expect(config.CloudWatch).ToNot(BeNil())

			Expect(config.CloudWatch.CloudWatchLogRoleArn).To(Equal("arn"))
			Expect(config.CloudWatch.CloudWatchLogGroupName).To(Equal("abcd"))
			Expect(config.CloudWatch.Applications).To(Equal([]string{"test3", "test4"}))
			Expect(config.CloudWatch.GroupsLogVersions).To(ContainElement("group-name"))

			Expect(config.S3.S3ConfigBucketName).To(Equal("foo"))
			Expect(config.S3.S3ConfigBucketPrefix).To(Equal("bar"))
			Expect(config.S3.Applications).To(Equal([]string{"test3", "test4"}))
			Expect(config.S3.GroupsLogVersions).To(ContainElement("group-name2"))
		})
	})
})

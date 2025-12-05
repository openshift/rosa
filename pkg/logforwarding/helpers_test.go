package logforwarding

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/reporter"
)

var testConfig = `
cloud_watch_log_role_arn: "arn"
cloud_watch_log_group_name: "abcd"
applications: ["test3", "test4"]
groups_log_version: ["group-name", "group-name2"]
s3_config_bucket_name: "foo"
s3_config_bucket_prefix: "bar"
`

func generateLogForwarderGroupVersions() []*cmv1.LogForwarderGroupVersions {
	test1, err := cmv1.NewLogForwarderGroupVersions().ID("Test1").Versions(
		cmv1.NewLogForwarderGroupVersion().ID("v1").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v2").Applications("4", "5", "6"),
		cmv1.NewLogForwarderGroupVersion().ID("v3").Applications("1", "2", "3")).Build()
	Expect(err).NotTo(HaveOccurred())
	test2, err := cmv1.NewLogForwarderGroupVersions().ID("Test2").Versions(
		cmv1.NewLogForwarderGroupVersion().ID("v1").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v2").Applications("0", "0", "0"),
		cmv1.NewLogForwarderGroupVersion().ID("v3").Applications("4", "5", "6")).Build()
	Expect(err).NotTo(HaveOccurred())
	test3, err := cmv1.NewLogForwarderGroupVersions().ID("Test3").Versions(
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

	Context("constructPodGroupsHelpMessage", func() {
		It("OK: Prints what is expected from a 3 entry long map", func() {
			Expect(constructPodGroupsHelpMessage(generateLogForwarderGroupVersions())).To(
				Equal("Test1: 1,2,3\nTest2: 4,5,6\nTest3: 7,8,9\n"),
			)
		})
	})

	Context("constructPodGroupsInteractiveOptions", func() {
		It("OK: Parses correctly", func() {
			Expect(constructPodGroupsInteractiveOptions(generateLogForwarderGroupVersions())).To(
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

		// bindCloudWatchLogForwarder and bindS3LogForwarder are also tested here
		It("OK: Unmarshals correctly", func() {
			r := reporter.CreateReporter()
			yamlPath := filepath.Join(tmpDir, "template")
			err := os.WriteFile(yamlPath, []byte(testConfig), 0644)
			Expect(err).ToNot(HaveOccurred())

			result, err := UnmarshalLogForwarderConfigYaml(r, yamlPath)
			Expect(len(result)).To(Equal(2))

			cloudWatchConfig := result[0]
			s3Config := result[1]

			Expect(cloudWatchConfig.CloudWatchLogRoleArn).To(Equal("arn"))
			Expect(cloudWatchConfig.CloudWatchLogGroupName).To(Equal("abcd"))
			Expect(cloudWatchConfig.Applications).To(Equal([]string{"test3", "test4"}))
			Expect(cloudWatchConfig.GroupsLogVersion).To(Equal([]string{"group-name", "group-name2"}))

			Expect(s3Config.S3ConfigBucketName).To(Equal("foo"))
			Expect(s3Config.S3ConfigBucketPrefix).To(Equal("bar"))
			Expect(s3Config.Applications).To(Equal([]string{"test3", "test4"}))
			Expect(s3Config.GroupsLogVersion).To(Equal([]string{"group-name", "group-name2"}))
		})
	})
})

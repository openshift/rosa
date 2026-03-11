package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
)

var _ = Describe("New Operator Iam Role From Cmv1", func() {
	const (
		fakeOperatorRoleArn = "arn:aws:iam::765374464689:role/fake-arn-openshift-cluster-csi-drivers-ebs-cloud-credentials"
	)
	It("OK: Converts cmv1 operator iam role", func() {
		cmv1OperatorIamRole, err := cmv1.NewOperatorIAMRole().
			Name("openshift").Namespace("operator").RoleARN(fakeOperatorRoleArn).Build()
		Expect(err).NotTo(HaveOccurred())
		ocmOperatorIamRole, err := NewOperatorIamRoleFromCmv1(cmv1OperatorIamRole)
		Expect(err).NotTo(HaveOccurred())
		Expect(ocmOperatorIamRole.Name).To(Equal(cmv1OperatorIamRole.Name()))
		Expect(ocmOperatorIamRole.Namespace).To(Equal(cmv1OperatorIamRole.Namespace()))
		Expect(ocmOperatorIamRole.RoleARN).To(Equal(cmv1OperatorIamRole.RoleARN()))
		path, err := aws.GetPathFromARN(cmv1OperatorIamRole.RoleARN())
		Expect(err).NotTo(HaveOccurred())
		Expect(ocmOperatorIamRole.Path).To(Equal(path))
	})
})

var _ = Context("Generate a query", func() {
	Describe("getClusterFilter", func() {
		It("Should return the default value", func() {
			output := getClusterFilter(nil)
			Expect(output).To(Equal("product.id = 'rosa'"))
		})

		It("Should construct a proper string if creator is not nil", func() {
			creator := &aws.Creator{AccountID: "test-account-id"}
			output := getClusterFilter(creator)
			Expect(output).To(Equal(
				"product.id = 'rosa' AND (properties.rosa_creator_arn LIKE 'arn:%:test-account-id:%' OR " +
					"aws.sts.role_arn LIKE 'arn:%:test-account-id:%')"))
		})
	})

})

var _ = Context("List Clusters", func() {
	Describe("List Clusters using Account Role", func() {

		creator := &aws.Creator{
			AccountID: "12345678",
		}
		role := aws.Role{}

		It("Correctly builds the query for Installer Account Role", func() {

			role.RoleType = aws.InstallerAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Installer-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE 'arn:%:12345678:%' OR aws.sts.role_arn LIKE 'arn:%:12345678:%') AND " +
				"aws.sts.role_arn='arn:aws:iam::765374464689:role/test-Installer-Role'"))
		})

		It("Correctly builds the query for Support Account Role", func() {

			role.RoleType = aws.SupportAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Support-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE 'arn:%:12345678:%' OR aws.sts.role_arn LIKE 'arn:%:12345678:%') AND " +
				"aws.sts.support_role_arn='arn:aws:iam::765374464689:role/test-Support-Role'"))
		})

		It("Correctly builds the query for Control Plane Account Role", func() {

			role.RoleType = aws.ControlPlaneAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-ControlPlane-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE 'arn:%:12345678:%' OR aws.sts.role_arn LIKE 'arn:%:12345678:%') AND " +
				"aws.sts.instance_iam_roles.master_role_arn='arn:aws:iam::765374464689:role/test-ControlPlane-Role'"))
		})

		It("Correctly builds the query for Worker Account Role", func() {

			role.RoleType = aws.WorkerAccountRoleType
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Worker-Role"

			query, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).NotTo(HaveOccurred())
			Expect(query).To(Equal("product.id = 'rosa' AND " +
				"(properties.rosa_creator_arn LIKE 'arn:%:12345678:%' OR aws.sts.role_arn LIKE 'arn:%:12345678:%') AND " +
				"aws.sts.instance_iam_roles.worker_role_arn='arn:aws:iam::765374464689:role/test-Worker-Role'"))
		})

		It("Fails to build query for unknown Role Type", func() {
			role.RoleType = "foo"
			role.RoleARN = "arn:aws:iam::765374464689:role/test-Worker-Role"

			_, err := getAccountRoleClusterFilter(creator, role)
			Expect(err).To(HaveOccurred())
		})

	})
})

var _ = Describe("Helper Functions", func() {
	Describe("buildVersion", func() {
		var config Spec
		var builder *cmv1.ClusterBuilder
		BeforeEach(func() {
			builder = cmv1.NewCluster()
			config = Spec{}
		})
		When("Version is specified", func() {
			BeforeEach(func() {
				config.Version = "openshift-version"
			})
			When("Channel is specified", func() {
				BeforeEach(func() {
					config.Channel = "stable-4.20"
				})
				When("ChannelGroup is specified", func() {
					BeforeEach(func() {
						config.ChannelGroup = "stable"
					})
					It("Should populate Channel but not ChannelGroup", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						Expect(cluster.Channel()).To(Equal("stable-4.20"))
						Expect(cluster.Version().ID()).To(Equal("openshift-version"))
						_, set := cluster.Version().GetChannelGroup()
						Expect(set).To(BeFalseBecause("ChannelGroup should not be set"))
					})
				})
				When("ChannelGroup is not specified", func() {
					It("Should populate Channel (and not ChannelGroup)", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						Expect(cluster.Channel()).To(Equal("stable-4.20"))
						Expect(cluster.Version().ID()).To(Equal("openshift-version"))
						_, set := cluster.Version().GetChannelGroup()
						Expect(set).To(BeFalseBecause("ChannelGroup should not be set"))
					})
				})
			})
			When("Channel is not specified", func() {
				When("ChannelGroup is specified", func() {
					BeforeEach(func() {
						config.ChannelGroup = "stable"
					})
					It("Should populate ChannelGroup and Channel be empty", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						Expect(cluster.Version().ChannelGroup()).To(Equal("stable"))
						Expect(cluster.Version().ID()).To(Equal("openshift-version"))
						_, set := cluster.GetChannel()
						Expect(set).To(BeFalseBecause("cluster Channel should not be set"))
					})
				})
				When("ChannelGroup is not specified", func() {
					It("Should populate Channel and ChannelGroup as empty", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						Expect(cluster.Version().ID()).To(Equal("openshift-version"))
						_, set := cluster.Version().GetChannelGroup()
						Expect(set).To(BeFalseBecause("ChannelGroup should not be set"))
						_, set = cluster.GetChannel()
						Expect(set).To(BeFalseBecause("Channel should not be set"))
					})
				})
			})
		})
		When("Version is not specified", func() {
			When("Channel is specified", func() {
				BeforeEach(func() {
					config.Channel = "stable-4.20"
				})
				When("ChannelGroup is specified", func() {
					BeforeEach(func() {
						config.ChannelGroup = "stable"
					})
					It("Should not populate ChannelGroup, and populate Channel", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						Expect(cluster.Channel()).To(Equal("stable-4.20"))
						_, set := cluster.GetVersion()
						Expect(set).To(BeFalseBecause("version should be completely unset"))
					})
				})
				When("ChannelGroup is not specified", func() {
					It("Should not populate ChannelGroup, and populate Channel", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						Expect(cluster.Channel()).To(Equal("stable-4.20"))
						_, set := cluster.GetVersion()
						Expect(set).To(BeFalseBecause("version should be completely unset"))
					})
				})
			})
			When("Channel is not specified", func() {
				When("ChannelGroup is specified", func() {
					BeforeEach(func() {
						config.ChannelGroup = "stable"
					})
					It("Should populate ChannelGroup, but not Channel", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						version, set := cluster.GetVersion()
						Expect(set).To(BeTrueBecause("version subobject should be present"))
						_, set = version.GetID()
						Expect(set).To(BeFalseBecause("version ID should not be set"))
						Expect(version.ChannelGroup()).To(Equal("stable"))
						_, set = cluster.GetChannel()
						Expect(set).To(BeFalseBecause("cluster Channel should not be set"))
					})
				})
				When("ChannelGroup is not specified", func() {
					It("Should not populate ChannelGroup or Channel", func() {
						Expect(buildVersion(config, builder)).Error().NotTo(HaveOccurred())
						cluster, err := builder.Build()
						Expect(err).NotTo(HaveOccurred())
						_, set := cluster.GetVersion()
						Expect(set).To(BeFalseBecause("version should be completely unset"))
						_, set = cluster.GetChannel()
						Expect(set).To(BeFalseBecause("cluster Channel should not be set"))
					})
				})
			})

		})
	})
})

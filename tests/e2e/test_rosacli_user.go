package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/ci/labels"
	"github.com/openshift/rosa/tests/utils/config"
	"github.com/openshift/rosa/tests/utils/exec/rosacli"
)

var _ = Describe("Edit User",
	labels.Day2,
	labels.FeatureUser,
	func() {
		defer GinkgoRecover()

		var (
			clusterID   string
			rosaClient  *rosacli.Client
			userService rosacli.UserService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			userService = rosaClient.User
		})

		AfterEach(func() {
			By("Clean remaining resources")
			err := rosaClient.CleanResources(clusterID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("can grant/list/revoke users - [id:36128]",
			labels.Critical,
			func() {
				var (
					dedicatedAdminsGroupName = "dedicated-admins"
					clusterAdminsGroupName   = "cluster-admins"
					dedicatedAdminsUserName  = "testdu"
					clusterAdminsUserName    = "testcu"
				)

				By("Try to list the user when there is no one")
				_, output, err := userService.ListUsers(clusterID)
				Expect(err).ToNot(HaveOccurred())
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(textData).To(ContainSubstring("INFO: There are no users configured for cluster '%s'", clusterID))

				By("Grant dedicated-admins user")
				out, err := userService.GrantUser(
					clusterID,
					dedicatedAdminsGroupName,
					dedicatedAdminsUserName,
				)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Granted role '%s' to user '%s' on cluster '%s'", dedicatedAdminsGroupName, dedicatedAdminsUserName, clusterID))

				By("Grant cluster-admins user")
				out, err = userService.GrantUser(
					clusterID,
					clusterAdminsGroupName,
					clusterAdminsUserName,
				)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Granted role '%s' to user '%s' on cluster '%s'", clusterAdminsGroupName, clusterAdminsUserName, clusterID))

				By("Get specific users")
				usersList, _, err := userService.ListUsers(clusterID)
				Expect(err).ToNot(HaveOccurred())

				user, err := usersList.User(dedicatedAdminsUserName)
				Expect(err).ToNot(HaveOccurred())
				Expect(user).NotTo(BeNil())
				Expect(user.Groups).To(Equal(dedicatedAdminsGroupName))

				user, err = usersList.User(clusterAdminsUserName)
				Expect(err).ToNot(HaveOccurred())
				Expect(user).NotTo(BeNil())
				Expect(user.Groups).To(Equal(clusterAdminsGroupName))

				By("Revoke dedicated-admins user")
				out, err = userService.RevokeUser(
					clusterID,
					dedicatedAdminsGroupName,
					dedicatedAdminsUserName,
				)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Revoked role '%s' from user '%s' on cluster '%s'", dedicatedAdminsGroupName, dedicatedAdminsUserName, clusterID))

				By("Revoke cluster-admins user")
				out, err = userService.RevokeUser(
					clusterID,
					clusterAdminsGroupName,
					clusterAdminsUserName,
				)
				Expect(err).ToNot(HaveOccurred())
				textData = rosaClient.Parser.TextData.Input(out).Parse().Tip()
				Expect(textData).Should(ContainSubstring("Revoked role '%s' from user '%s' on cluster '%s'", clusterAdminsGroupName, clusterAdminsUserName, clusterID))

				By("List users after revoke")
				usersList, _, err = userService.ListUsers(clusterID)
				// Comment this part due to known issue
				// Expect(err).ToNot(HaveOccurred())

				foundUser, err := usersList.User(dedicatedAdminsUserName)
				Expect(err).ToNot(HaveOccurred())
				Expect(foundUser).To(Equal(rosacli.GroupUser{}))

				foundUser, err = usersList.User(clusterAdminsUserName)
				Expect(err).ToNot(HaveOccurred())
				Expect(foundUser).To(Equal(rosacli.GroupUser{}))
			})
	})

var _ = Describe("Validate user",
	labels.Day2, // TODO could be transformed as day1 negative
	labels.FeatureUser,
	func() {
		defer GinkgoRecover()
		var (
			invalidPassword = "password1" // disallowed password
			validPassword   = "Th3long,validpassword"
			clusterID       string

			rosaClient     *rosacli.Client
			clusterService rosacli.ClusterService
		)

		BeforeEach(func() {
			By("Get the cluster")
			clusterID = config.GetClusterID()
			Expect(clusterID).ToNot(Equal(""), "ClusterID is required. Please export CLUSTER_ID")

			By("Init the client")
			rosaClient = rosacli.NewClient()
			clusterService = rosaClient.Cluster
		})

		It("try to create cluster with invalid usernames, passwords or unsupported configurations - [id:66362]",
			labels.Critical,
			func() {
				clusterID = "fake-cluster" // these tests do not create or use a real cluster so no need to address an existing one.

				By("Try to create classic non STS cluster with invalid admin password")
				output, err := clusterService.CreateDryRun(clusterID, "--cluster-admin-password", invalidPassword,
					"--region", "us-east-2", "--mode", "auto", "-y")
				textData := rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(err).To(HaveOccurred())
				Expect(textData).Should(ContainSubstring("assword must be at least"))

				By("Try to create cluster with invalid admin password on classic STS cluster")
				output, err = clusterService.CreateDryRun(clusterID, "--sts", "--cluster-admin-password", invalidPassword,
					"--region", "us-east-2", "--mode", "auto", "-y")
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(err).To(HaveOccurred())
				Expect(textData).Should(ContainSubstring("assword must be at least"))

				By("Try to create Hypershift cluster with admin username and password set (unsupported)")
				output, err = clusterService.CreateDryRun(clusterID, "--hosted-cp",
					"--cluster-admin-password", validPassword,
					"--region", "us-west-2",
					"--support-role-arn", "fake",
					"--controlplane-iam-role", "fake",
					"--worker-iam-role", "fake",
					"--mode", "auto", "-y")
				textData = rosaClient.Parser.TextData.Input(output).Parse().Tip()
				Expect(err).To(HaveOccurred())
				Expect(textData).Should(ContainSubstring("is only supported in classic"))
			})

	})

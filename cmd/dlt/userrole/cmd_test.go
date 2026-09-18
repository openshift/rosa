package userrole

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/test"
)

const (
	testRoleARN  = "arn:aws:iam::123456789012:role/ManagedOpenShift-User-testuser-Role"
	testRoleName = "ManagedOpenShift-User-testuser-Role"
)

var _ = Describe("delete user-role", func() {
	var t *test.TestingRuntime

	BeforeEach(func() {
		t = test.NewTestRuntime()
		t.RosaRuntime.Creator = &aws.Creator{
			AccountID: "123456789012",
			Partition: "aws",
		}
		args = struct {
			roleARN string
		}{}
		interactive.SetEnabled(false)
		interactive.SetModeKey("")
		Expect(Cmd.Flags().Set("mode", "")).To(Succeed())
	})

	Context("runWithRuntime", func() {
		It("returns error when mode is invalid", func() {
			interactive.SetModeKey("invalid_mode")
			Expect(Cmd.Flags().Set("mode", "invalid_mode")).To(Succeed())

			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Invalid mode"))
		})

		It("returns error when role ARN is empty", func() {
			err := runWithRuntime(t.RosaRuntime, Cmd)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid user role ARN to delete from the current AWS account"))
		})
	})

	Context("buildCommands", func() {
		It("generates detach and delete role commands", func() {
			mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
			policyARN := "arn:aws:iam::123456789012:policy/ManagedOpenShift-User-testuser-Policy"
			roleName := testRoleName

			mockClient.EXPECT().GetAttachedPolicy(&roleName).Return([]aws.PolicyDetail{
				{PolicyArn: policyARN},
			}, nil)
			mockClient.EXPECT().HasPermissionsBoundary(testRoleName).Return(false, nil)

			commands, err := buildCommands(testRoleName, testRoleARN, false, mockClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(commands).To(ContainSubstring("detach-role-policy"))
			Expect(commands).To(ContainSubstring(policyARN))
			Expect(commands).To(ContainSubstring("delete-role"))
			Expect(commands).To(ContainSubstring(testRoleName))
		})

		It("includes unlink command when role is linked", func() {
			mockClient := t.RosaRuntime.AWSClient.(*aws.MockClient)
			roleName := testRoleName

			mockClient.EXPECT().GetAttachedPolicy(&roleName).Return([]aws.PolicyDetail{}, nil)
			mockClient.EXPECT().HasPermissionsBoundary(testRoleName).Return(false, nil)

			commands, err := buildCommands(testRoleName, testRoleARN, true, mockClient)
			Expect(err).NotTo(HaveOccurred())
			Expect(commands).To(ContainSubstring("rosa unlink user-role"))
			Expect(commands).To(ContainSubstring(testRoleARN))
		})
	})
})

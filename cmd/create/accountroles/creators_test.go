package accountroles

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/ginkgo/v2/dsl/decorators"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Accountroles", Ordered, func() {
	When("createRoles", func() {
		It("createRole fails to find policy ARN", func() {
			mockCtrl := gomock.NewController(GinkgoT())
			mockClient := mock.NewMockClient(mockCtrl)
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("role-123", nil)

			r := rosa.NewRuntime()
			r.AWSClient = mockClient
			r.Creator = &mock.Creator{ARN: "arn-123"}
			policies := map[string]*cmv1.AWSSTSPolicy{}
			accountRolesCreationInput := buildRolesCreationInput("test", "", "account-123", "stage", policies, "", "")
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("failed to find policy ARN for 'sts_hcp_installer_permission_policy'"))
		})
		It("createRole succeeds", func() {
			mockCtrl := gomock.NewController(GinkgoT())
			mockClient := mock.NewMockClient(mockCtrl)
			mockClient.EXPECT().AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			mockClient.EXPECT().EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return("role-123", nil).AnyTimes()

			r := rosa.NewRuntime()
			r.AWSClient = mockClient
			r.Creator = &mock.Creator{ARN: "arn-123"}
			installerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::installer").Build()
			workerPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::worker").Build()
			supportPolicy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::support").Build()
			ec2Policy, _ := (&cmv1.AWSSTSPolicyBuilder{}).ARN("arn::ec2").Build()

			policies := map[string]*cmv1.AWSSTSPolicy{
				"sts_hcp_installer_permission_policy":       installerPolicy,
				"sts_hcp_instance_worker_permission_policy": workerPolicy,
				"sts_hcp_support_permission_policy":         supportPolicy,
				"sts_hcp_ec2_registry_permission_policy":    ec2Policy,
			}

			accountRolesCreationInput := buildRolesCreationInput("test", "", "account-123", "stage", policies, "", "")
			err := (&hcpManagedPoliciesCreator{}).createRoles(r, accountRolesCreationInput)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

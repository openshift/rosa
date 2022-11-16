package roles_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift/rosa/pkg/helper/roles"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Roles helper", func() {
	var _ = Describe("Validates Random Label function", func() {
		var _ = Context("when generating random labels", func() {

			It("Retrieves operator role name from populated cluster given a STSOperator", func() {
				clusterBuilder := cmv1.NewCluster().
					AWS(cmv1.NewAWS().STS(cmv1.NewSTS().OperatorIAMRoles(cmv1.NewOperatorIAMRole().RoleARN("arn:aws:iam::111111111111:role/testprefix-openshift-namespace-name").Namespace("openshift-namespace").Name("name"))))
				cluster, _ := clusterBuilder.Build()
				newSTSOperator, _ := cmv1.NewSTSOperator().Namespace("openshift-namespace2").Name("new").Build()
				operatorRoleName := GetOperatorRoleName(cluster, newSTSOperator)
				Expect("testprefix-openshift-namespace2-new").To(Equal(operatorRoleName))
			})

			It("Returns empty error invalid arn", func() {
				clusterBuilder := cmv1.NewCluster().
					AWS(cmv1.NewAWS().STS(cmv1.NewSTS().OperatorIAMRoles(cmv1.NewOperatorIAMRole().RoleARN("testprefix-openshift-namespace-name").Namespace("openshift-namespace").Name("name"))))
				cluster, _ := clusterBuilder.Build()
				newSTSOperator, _ := cmv1.NewSTSOperator().Namespace("openshift-namespace2").Name("new").Build()
				operatorRoleName := GetOperatorRoleName(cluster, newSTSOperator)
				Expect("").To(Equal(operatorRoleName))
			})

			It("Returns empty alongside operator due to missing -openshift", func() {
				clusterBuilder := cmv1.NewCluster().
					AWS(cmv1.NewAWS().STS(cmv1.NewSTS().OperatorIAMRoles(cmv1.NewOperatorIAMRole().RoleARN("arn:aws:iam::111111111111:role/testprefix-namespace-name").Namespace("namespace").Name("name"))))
				cluster, _ := clusterBuilder.Build()
				newSTSOperator, _ := cmv1.NewSTSOperator().Namespace("openshift-namespace2").Name("new").Build()
				operatorRoleName := GetOperatorRoleName(cluster, newSTSOperator)
				Expect("-openshift-namespace2-new").To(Equal(operatorRoleName))
			})
		})

	})
})

package aws_test

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/iam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	. "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
)

const (
	testArn = "arn:aws:iam::123456789012:role/xxx-OCM-Role-1223456778"
)

var _ = Describe("Helpers", func() {
	var _ = Describe("Validates Jump Accounts", func() {
		var _ = Context("when different envs", func() {

			It("Retrieves production jump account", func() {
				Expect("710019948333").To(Equal(GetJumpAccount("production")))
			})

			It("Retrieves staging jump account", func() {
				Expect("644306948063").To(Equal(GetJumpAccount("staging")))
			})

			It("Retrieves integration jump account", func() {
				Expect("896164604406").To(Equal(GetJumpAccount("integration")))
			})

			It("Retrieves local jump account", func() {
				Expect("765374464689").To(Equal(GetJumpAccount("local")))
			})

			It("Retrieves local-proxy jump account", func() {
				Expect("765374464689").To(Equal(GetJumpAccount("local-proxy")))
			})

			It("Retrieves crc jump account", func() {
				Expect("765374464689").To(Equal(GetJumpAccount("crc")))
			})
		})
	})

	var _ = Describe("Validates ARNValidator", func() {
		var _ = Context("when valid arn", func() {
			It("Parses valid arn", func() {
				err := ARNValidator(testArn)
				Expect(err).To(Not(HaveOccurred()))
			})
		})

		var _ = Context("when invalid arn", func() {
			It("Produces non string error", func() {
				err := ARNValidator(1)
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "can only validate strings")))
			})

			It("Produces invalid arn error", func() {
				err := ARNValidator("test")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "Invalid ARN")))
			})
		})
	})

	var _ = Describe("Validates ARNPathValidator", func() {
		var _ = Context("when valid arn path", func() {
			It("Matches valid arn path", func() {
				err := ARNPathValidator("/testpath/")
				Expect(err).To(Not(HaveOccurred()))
			})
		})

		var _ = Context("when invalid arn path", func() {
			It("Produces non string error", func() {
				err := ARNPathValidator(1)
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "can only validate strings")))
			})

			It("Produces invalid arn path error", func() {
				err := ARNPathValidator("test")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "invalid ARN Path")))
			})

			It("Produces invalid arn path error", func() {
				err := ARNPathValidator("/test")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "invalid ARN Path.")))
			})

			It("Produces invalid arn path error", func() {
				err := ARNPathValidator("test/")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "invalid ARN Path.")))
			})

			It("Produces invalid arn path error", func() {
				err := ARNPathValidator("/@test/")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "invalid ARN Path.")))
			})
		})
	})

	var _ = Describe("Validates GetRegion", func() {
		var _ = Context("when region param filled", func() {
			It("Retrieves user input", func() {
				region, err := GetRegion("region")
				Expect(err).ToNot(HaveOccurred())
				Expect("region").To(Equal(region))
			})
		})

		var _ = Context("when region param empty", func() {
			It("Retrieves config region", func() {
				region, err := GetRegion("")
				Expect(err).ToNot(HaveOccurred())
				Expect("us-east-1").To(Equal(region))
			})
		})
	})

	var _ = Describe("Validates GetRegion", func() {
		var _ = Context("when region param filled", func() {
			It("Retrieves user input", func() {
				region, err := GetRegion("region")
				Expect(err).ToNot(HaveOccurred())
				Expect("region").To(Equal(region))
			})
		})

		var _ = Context("when region param empty", func() {
			It("Retrieves config region", func() {
				region, err := GetRegion("")
				Expect(err).ToNot(HaveOccurred())
				Expect("us-east-1").To(Equal(region))
			})
		})
	})

	var _ = Describe("Validates UserTagValidator", func() {
		var _ = Context("when valid input", func() {
			It("Produces nil", func() {
				Expect(UserTagValidator("")).To(BeNil())
			})

			It("Produces nil", func() {
				Expect(UserTagValidator("foo:bar,bar:foo")).To(BeNil())
			})
		})

		var _ = Context("when invalid input", func() {
			It("Produces 'can only validate strings'", func() {
				err := UserTagValidator(1)
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "can only validate strings")))
			})

			It("Produces 'invalid tag format'", func() {
				err := UserTagValidator("foo=bar,bar=foo")
				Expect(err).To(HaveOccurred())
				Expect("invalid tag format, Tags are comma separated, for example: --tags=foo:bar,bar:baz").
					To(Equal(fmt.Sprintf("%s", err)))
			})

			It("Produces 'invalid tag format'", func() {
				err := UserTagValidator("foo:bar:baz,bar:foo")
				Expect(err).To(HaveOccurred())
				Expect("invalid tag format. Expected tag format: --tags=key:value").
					To(Equal(fmt.Sprintf("%s", err)))
			})

			It("Produces 'key regex match error'", func() {
				err := UserTagValidator(":bar")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "expected a valid user tag key")))
			})

			It("Produces 'value regex match error'", func() {
				err := UserTagValidator("foo:" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" +
					"aaaaaaa")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "expected a valid user tag value")))
			})
		})
	})

	var _ = Describe("Validates UserNoProxyValidator", func() {
		var _ = Context("when valid input", func() {
			It("Produces nil", func() {
				Expect(UserNoProxyValidator("")).To(BeNil())
			})

			It("Produces nil", func() {
				Expect(UserNoProxyValidator("example.com,example2.com")).To(BeNil())
			})
		})

		var _ = Context("when invalid input", func() {
			It("Produces 'can only validate strings'", func() {
				err := UserNoProxyValidator(1)
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "can only validate strings")))
			})

			It("Produces 'user tag keys must be unique'", func() {
				err := UserNoProxyValidator("foo:bar,foo:baz")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "expected a valid user no-proxy value")))
			})
		})
	})

	var _ = Describe("Validates UserNoProxyDuplicateValidator", func() {
		var _ = Context("when valid input", func() {
			It("Produces nil", func() {
				Expect(UserNoProxyDuplicateValidator("")).To(BeNil())
			})

			It("Produces nil", func() {
				Expect(UserNoProxyDuplicateValidator("example.com,example2.com")).To(BeNil())
			})
		})

		var _ = Context("when invalid input", func() {
			It("Produces 'can only validate strings'", func() {
				err := UserNoProxyDuplicateValidator(1)
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "can only validate strings")))
			})

			It("Produces 'user tag keys must be unique'", func() {
				err := UserNoProxyDuplicateValidator("example.com,example.com")
				Expect(err).To(HaveOccurred())
				Expect(true).To(Equal(strings.Contains(fmt.Sprintf("%s", err), "no-proxy values must be unique")))
			})
		})
	})

	var _ = Describe("Validates GetTagValues", func() {

		roleType := tags.RoleType
		roleValue := "installer"
		openShiftVersion := tags.OpenShiftVersion
		openShiftVersionValue := "4.11"

		It("Retrieves no values", func() {
			empty := []*iam.Tag{}
			roleType, version := GetTagValues(empty)
			Expect("").To(Equal(roleType))
			Expect("").To(Equal(version))
		})

		It("Retrieves one value roleType", func() {
			iamTags := []*iam.Tag{{Key: &roleType, Value: &roleValue}}
			roleType, version := GetTagValues(iamTags)
			Expect("installer").To(Equal(roleType))
			Expect("").To(Equal(version))
		})

		It("Retrieves one value openshiftVersion", func() {
			iamTags := []*iam.Tag{{Key: &openShiftVersion, Value: &openShiftVersionValue}}
			roleType, version := GetTagValues(iamTags)
			Expect("4.11").To(Equal(version))
			Expect("").To(Equal(roleType))
		})

		It("Retrieves both values", func() {
			iamTags := []*iam.Tag{{Key: &roleType, Value: &roleValue},
				{Key: &openShiftVersion, Value: &openShiftVersionValue}}
			roleType, version := GetTagValues(iamTags)
			Expect("installer").To(Equal(roleType))
			Expect("4.11").To(Equal(version))
		})
	})

	var _ = Describe("Validates GetRoleName", func() {
		role := "Installer"

		It("Retrieves role name", func() {
			roleName := GetRoleName("a", role)
			Expect("a-Installer-Role").To(Equal(roleName))
		})

		It("Retrieves truncated role name", func() {
			roleName := GetRoleName("aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaa", role)
			Expect("aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaa").To(Equal(roleName))
		})
	})

	var _ = Describe("Validates GetOCMRoleName", func() {
		externalID := "123456789"

		It("Retrieves role name", func() {
			roleName := GetOCMRoleName("a", externalID)
			Expect("a-OCM-Role-123456789").To(Equal(roleName))
		})

		It("Retrieves truncated role name", func() {
			roleName := GetOCMRoleName("aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaa", externalID)
			Expect("aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaa").To(Equal(roleName))
		})
	})

	var _ = Describe("Validates GetOCMRoleName", func() {
		testName := "test"

		It("Retrieves role name", func() {
			roleName := GetUserRoleName("a", testName)
			Expect("a-User-test-Role").To(Equal(roleName))
		})

		It("Retrieves truncated role name", func() {
			roleName := GetUserRoleName("aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaa", testName)
			Expect("aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaa").To(Equal(roleName))
		})
	})

	var _ = Describe("Validates GetOperatorPolicyName", func() {
		namespace := "openshift"
		name := "cloud-control"

		It("Retrieves policy name", func() {
			policyName := GetOperatorPolicyName("a", namespace, name)
			Expect("a-openshift-cloud-control").To(Equal(policyName))
		})

		It("Retrieves truncated policy name", func() {
			policyName := GetOperatorPolicyName("aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaaaaaaa"+
				"aaaaa", namespace, name)
			Expect("aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaaaaaaaa" +
				"aaaa").To(Equal(policyName))
		})
	})

	var _ = Describe("Validates GetAdminPolicyName", func() {
		It("Retrieves admin policy name", func() {
			policyName := GetAdminPolicyName("a")
			Expect("a-Admin-Policy").To(Equal(policyName))
		})
	})

	var _ = Describe("Validates GetPolicyName", func() {
		It("Retrieves policy name", func() {
			policyName := GetPolicyName("a")
			Expect("a-Policy").To(Equal(policyName))
		})
	})

	var _ = Describe("Validates GetOperatorPolicyARN", func() {
		namespace := "openshift"
		name := "cloud-control"

		It("Retrieves operator policy arn without path", func() {
			operatorPolicyARN := GetOperatorPolicyARN("123", "a", namespace, name, "")
			Expect("arn:aws:iam::123:policy/a-openshift-cloud-control").To(Equal(operatorPolicyARN))
		})

		It("Retrieves operator policy arn with path", func() {
			operatorPolicyARN := GetOperatorPolicyARN("123", "a", namespace, name, "/path/")
			Expect("arn:aws:iam::123:policy/path/a-openshift-cloud-control").To(Equal(operatorPolicyARN))
		})
	})

	var _ = Describe("Validates GetAdminPolicyARN", func() {

		It("Retrieves admin policy arn without path", func() {
			adminPolicyARN := GetAdminPolicyARN("123", "a", "")
			Expect("arn:aws:iam::123:policy/a-Admin-Policy").To(Equal(adminPolicyARN))
		})

		It("Retrieves admin policy arn with path", func() {
			adminPolicyARN := GetAdminPolicyARN("123", "a", "/path/")
			Expect("arn:aws:iam::123:policy/path/a-Admin-Policy").To(Equal(adminPolicyARN))
		})
	})

	var _ = Describe("Validates GetPolicyARN", func() {

		It("Retrieves policy arn without path", func() {
			policyARN := GetPolicyARN("123", "a", "")
			Expect("arn:aws:iam::123:policy/a-Policy").To(Equal(policyARN))
		})

		It("Retrieves policy arn with path", func() {
			policyARN := GetPolicyARN("123", "a", "/path/")
			Expect("arn:aws:iam::123:policy/path/a-Policy").To(Equal(policyARN))
		})
	})

	var _ = Describe("Validates GetPathFromAccountRole", func() {

		It("Retrieves path arn without path", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
				RoleARN("arn:aws:iam::123:role/ManagedOpenShift-Installer-Role").
				SupportRoleARN("arn:aws:iam::123:role/ManagedOpenShift-Support-Role").
				InstanceIAMRoles(cmv1.NewInstanceIAMRoles().
					MasterRoleARN("arn:aws:iam::123:role/ManagedOpenShift-ControlPlane-Role").
					WorkerRoleARN("arn:aws:iam::123:role/ManagedOpenShift-Worker-Role")))).
				Build()
			Expect(err).NotTo(HaveOccurred())
			for _, role := range AccountRoles {
				path, err := GetPathFromAccountRole(cluster, role.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect("").To(Equal(path))
			}
		})

		It("Retrieves path arn with path", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
				RoleARN("arn:aws:iam::123:role/path/ManagedOpenShift-Installer-Role").
				SupportRoleARN("arn:aws:iam::123:role/path/ManagedOpenShift-Support-Role").
				InstanceIAMRoles(cmv1.NewInstanceIAMRoles().
					MasterRoleARN("arn:aws:iam::123:role/path/ManagedOpenShift-ControlPlane-Role").
					WorkerRoleARN("arn:aws:iam::123:role/path/ManagedOpenShift-Worker-Role")))).
				Build()
			Expect(err).NotTo(HaveOccurred())
			for _, role := range AccountRoles {
				path, err := GetPathFromAccountRole(cluster, role.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect("/path/").To(Equal(path))
			}
		})
	})

	var _ = Describe("Validates GetPathFromARN", func() {

		var _ = Context("when valid input", func() {

			It("Retrieves path arn without path", func() {
				path, err := GetPathFromARN("arn:aws:iam::123:policy/a-Policy")
				Expect(err).NotTo(HaveOccurred())
				Expect("").To(Equal(path))
			})

			It("Retrieves path arn with path", func() {
				path, err := GetPathFromARN("arn:aws:iam::123:policy/path/a-Policy")
				Expect(err).NotTo(HaveOccurred())
				Expect("/path/").To(Equal(path))
			})
		})

		var _ = Context("when invalid input", func() {
			It("Retrieves path arn without path", func() {
				path, err := GetPathFromARN(":policy/a-Policy")
				Expect(err).To(HaveOccurred())
				Expect("arn: invalid prefix").To(Equal(fmt.Sprintf("%s", err)))
				Expect("").To(Equal(path))
			})
		})
	})

	var _ = Describe("Validates GetRoleARN", func() {

		It("Retrieves role arn without path", func() {
			role := GetRoleARN("123", "test", "")
			Expect("arn:aws:iam::123:role/test").To(Equal(role))
		})

		It("Retrieves role arn with path", func() {
			role := GetRoleARN("123", "test", "/path/")
			Expect("arn:aws:iam::123:role/path/test").To(Equal(role))
		})
	})

	var _ = Describe("Validates GetOIDCProviderARN", func() {

		It("Retrieves oidc arn", func() {
			oidc := GetOIDCProviderARN("123", "provider-test")
			Expect("arn:aws:iam::123:oidc-provider/provider-test").To(Equal(oidc))
		})
	})

	var _ = Describe("Validates GetPartition", func() {

		It("Retrieves oidc arn", func() {
			partion := GetPartition()
			Expect("aws").To(Equal(partion))
		})
	})

	var _ = Describe("Validates GetPrefixFromAccountRole", func() {

		It("Retrieves role prefix", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
				RoleARN("arn:aws:iam::123:role/ManagedOpenShift-Installer-Role").
				SupportRoleARN("arn:aws:iam::123:role/ManagedOpenShift-Support-Role").
				InstanceIAMRoles(cmv1.NewInstanceIAMRoles().
					MasterRoleARN("arn:aws:iam::123:role/ManagedOpenShift-ControlPlane-Role").
					WorkerRoleARN("arn:aws:iam::123:role/ManagedOpenShift-Worker-Role")))).
				Build()
			Expect(err).NotTo(HaveOccurred())
			for _, role := range AccountRoles {
				accRolePrefix, err := GetPrefixFromAccountRole(cluster, role.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect("ManagedOpenShift").To(Equal(accRolePrefix))
			}
		})

		It("Retrieves empty role prefix", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS()).Build()
			Expect(err).NotTo(HaveOccurred())
			for _, role := range AccountRoles {
				accRolePrefix, err := GetPrefixFromAccountRole(cluster, role.Name)
				Expect(err).NotTo(HaveOccurred())
				Expect("").To(Equal(accRolePrefix))
			}
		})
	})

	var _ = Describe("Validates GetPrefixFromInstallerAccountRole", func() {

		It("Retrieves role prefix from installer role", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
				RoleARN("arn:aws:iam::123:role/ManagedOpenShift-Installer-Role"))).
				Build()
			Expect(err).NotTo(HaveOccurred())
			accInstallerRolePrefix, err := GetPrefixFromInstallerAccountRole(cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect("ManagedOpenShift").To(Equal(accInstallerRolePrefix))
		})

		It("Retrieves empty prefix if installer role is not present", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
				SupportRoleARN("arn:aws:iam::123:role/ManagedOpenShift-Support-Role"))).
				Build()
			Expect(err).NotTo(HaveOccurred())
			accInstallerRolePrefix, err := GetPrefixFromInstallerAccountRole(cluster)
			Expect(err).NotTo(HaveOccurred())
			Expect("").To(Equal(accInstallerRolePrefix))
		})
	})

	var _ = Describe("Validates GetPrefixFromOperatorRole", func() {

		It("Retrieves role prefix from operator role", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
				OperatorIAMRoles(cmv1.NewOperatorIAMRole().
					RoleARN("arn:aws:iam::123:role/operator-test-prefix-openshift-namespace-name").
					Namespace("openshift-namespace").
					Name("name")))).
				Build()
			Expect(err).NotTo(HaveOccurred())
			operatorRolePrefix := GetPrefixFromOperatorRole(cluster)
			Expect("operator-test-prefix").To(Equal(operatorRolePrefix))
		})

		It("Retrieves empty prefix if operator role is not present", func() {
			cluster, err := cmv1.NewCluster().AWS(cmv1.NewAWS().STS(cmv1.NewSTS().
				OperatorIAMRoles())).Build()
			Expect(err).NotTo(HaveOccurred())
			operatorRolePrefix := GetPrefixFromOperatorRole(cluster)
			Expect("").To(Equal(operatorRolePrefix))
		})
	})

})

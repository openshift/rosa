package cluster

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/spf13/cobra"

	mock "github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/aws/tags"
	"github.com/openshift/rosa/pkg/logging"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var _ = Describe("Validate build command", func() {

	var clusterConfig ocm.Spec
	var operatorRolesPrefix string
	var expectedOperatorRolePath string
	var userSelectedAvailabilityZones bool
	var defaultMachinePoolLabels string
	var argsDotProperties []string

	BeforeEach(func() {
		clusterConfig = ocm.Spec{
			Name: "cluster-name",
		}
		operatorRolesPrefix = "prefix"
		expectedOperatorRolePath = "operator-role-path"
		userSelectedAvailabilityZones = false
		defaultMachinePoolLabels = "machine-pool-label"
	})
	Context("build command", func() {
		When("domain prefix is present", func() {
			It("prints --domain-prefix", func() {
				clusterConfig.DomainPrefix = "dns-label"
				command := buildCommand(clusterConfig, operatorRolesPrefix,
					expectedOperatorRolePath, userSelectedAvailabilityZones,
					defaultMachinePoolLabels, argsDotProperties)
				Expect(command).To(Equal(
					"rosa create cluster --cluster-name cluster-name --domain-prefix dns-label" +
						" --operator-roles-prefix prefix"))
			})
		})

		When("--etcd-encryption is true", func() {
			It("prints --etcd-encryption-kms-arn", func() {
				clusterConfig.EtcdEncryption = true
				clusterConfig.EtcdEncryptionKMSArn = "my-test-arn"
				command := buildCommand(clusterConfig, operatorRolesPrefix,
					expectedOperatorRolePath, userSelectedAvailabilityZones,
					defaultMachinePoolLabels, argsDotProperties)
				Expect(command).To(Equal(
					"rosa create cluster --cluster-name cluster-name --operator-roles-prefix prefix" +
						" --etcd-encryption --etcd-encryption-kms-arn my-test-arn"))
			})
		})

		When("--etcd-encryption is false", func() {
			It("Does not print --etc-encryption-kms-arn", func() {
				clusterConfig.EtcdEncryption = false
				clusterConfig.EtcdEncryptionKMSArn = "my-test-arn"
				command := buildCommand(clusterConfig, operatorRolesPrefix,
					expectedOperatorRolePath, userSelectedAvailabilityZones,
					defaultMachinePoolLabels, argsDotProperties)
				Expect(command).To(Equal(
					"rosa create cluster --cluster-name cluster-name --operator-roles-prefix prefix"))
			})
		})

		When("--properties is not present", func() {
			It("should not include --properties", func() {
				command := buildCommand(clusterConfig, operatorRolesPrefix,
					expectedOperatorRolePath, userSelectedAvailabilityZones,
					defaultMachinePoolLabels, argsDotProperties)
				// nolint:lll
				Expect(command).To(Equal("rosa create cluster --cluster-name cluster-name --operator-roles-prefix prefix"))
			})
		})
		When("--properties is present", func() {
			It("should include --properties", func() {
				argsDotProperties = []string{"prop1", "prop2"}
				command := buildCommand(clusterConfig, operatorRolesPrefix,
					expectedOperatorRolePath, userSelectedAvailabilityZones,
					defaultMachinePoolLabels, argsDotProperties)
				// nolint:lll
				Expect(command).To(Equal("rosa create cluster --cluster-name cluster-name --operator-roles-prefix prefix --properties \"prop1\" --properties \"prop2\""))
			})
		})

		When("--external-auth-providers-enabled is true", func() {
			It("prints --external-auth-providers-enabled", func() {
				args.externalAuthProvidersEnabled = true
				command := buildCommand(clusterConfig, operatorRolesPrefix,
					expectedOperatorRolePath, userSelectedAvailabilityZones,
					defaultMachinePoolLabels, argsDotProperties)
				Expect(command).To(Equal(
					"rosa create cluster --cluster-name cluster-name --operator-roles-prefix prefix" +
						" --external-auth-providers-enabled --properties \"prop1\" --properties \"prop2\""))
			})
		})
	})
	Context("build tags command", func() {
		When("tag key or values DO contain a colon", func() {
			It("should build tags command with a space as a delimiter", func() {
				tags := map[string]string{
					"key1":   "value1",
					"key2":   "value2",
					"key3:4": "value3:4",
					"key5":   "value5:6",
				}

				formattedTags := buildTagsCommand(tags)

				Expect(len(formattedTags)).To(Equal(len(tags)),
					"expected not to lose any tags while formatting")
				for _, tag := range formattedTags {
					if strings.Contains(tag, "key3") {
						Expect(strings.Contains(tag, ":")).To(Equal(true),
							"expected `:` to not be removed from key/value")
					}

					Expect(strings.Contains(tag, " ")).To(Equal(true),
						"expected delim to be ' '")

				}
			})
		})

		When("tag key or values DO NOT contain a colon", func() {
			It("should build tags command with default delimiter", func() {
				tags := map[string]string{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
					"key4": "value4",
					"key5": "value5",
				}

				formattedTags := buildTagsCommand(tags)

				Expect(len(formattedTags)).To(Equal(len(tags)),
					"expected not to lose any tags while formatting")
				for _, tag := range formattedTags {
					Expect(strings.Contains(tag, ":")).To(Equal(true),
						"expected delim to be ':'")

				}
			})
		})
	})
})

var _ = Describe("Validates OCP version", func() {

	const (
		nightly   = "nightly"
		stable    = "stable"
		candidate = "candidate"
		fast      = "fast"
	)
	var client *ocm.Client
	BeforeEach(func() {
		// todo this test expects and uses a real ocm client
		// disabling the test until we can mock this to run in prow
		Skip("disabling test until ocm client is mocked")
		c, err := ocm.NewClient().Logger(logging.NewLogger()).Build()
		Expect(err).NotTo(HaveOccurred())
		client = c
	})

	var _ = Context("when creating a hosted cluster", func() {

		It("OK: Validates successfully a cluster for hosted clusters with a supported version", func() {
			v, err := client.ValidateVersion("4.12.5", []string{"4.12.5"}, stable, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.12.5"))
		})

		It("OK: Validates successfully a nightly version of OCP for hosted clusters "+
			"with a supported version", func() {
			v, err := client.ValidateVersion("4.12.0-0.nightly-2023-04-10-222146",
				[]string{"4.12.0-0.nightly-2023-04-10-222146"}, nightly, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.12.0-0.nightly-2023-04-10-222146-nightly"))
		})

		It("KO: Fails with a nightly version of OCP for hosted clusters "+
			"in a not supported version", func() {
			v, err := client.ValidateVersion("4.11.0-0.nightly-2022-10-17-040259",
				[]string{"4.11.0-0.nightly-2022-10-17-040259"}, nightly, false, true)
			Expect(err).To(BeEquivalentTo(
				fmt.Errorf("version '4.11.0-0.nightly-2022-10-17-040259' " +
					"is not supported for hosted clusters")))
			Expect(v).To(Equal(""))
		})

		It("OK: Validates successfully the next major release of OCP for hosted clusters "+
			"with a supported version", func() {
			v, err := client.ValidateVersion("4.13.0-rc.2", []string{"4.13.0-rc.2"}, candidate, false, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.13.0-rc.2-candidate"))
		})

		It(`KO: Fails to validate a cluster for a hosted
		cluster when the user provides an unsupported version`, func() {
			v, err := client.ValidateVersion("4.11.5", []string{"4.11.5"}, stable, false, true)
			Expect(err).To(BeEquivalentTo(fmt.Errorf("version '4.11.5' is not supported for hosted clusters")))
			Expect(v).To(BeEmpty())
		})

		It(`KO: Fails to validate a cluster for a hosted cluster
		when the user provides an invalid or malformed version`, func() {
			v, err := client.ValidateVersion("foo.bar", []string{"foo.bar"}, stable, false, true)
			Expect(err).To(BeEquivalentTo(
				fmt.Errorf("version 'foo.bar' was not found")))
			Expect(v).To(BeEmpty())
		})

		It(`OK: Validates a supported Hypershift version successfully`, func() {
			v, err := client.ValidateHypershiftVersion("4.14.5", stable)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(BeTrue())
		})

		It(`KO: Fails to validate Hypershift version when the version is less than the minimal supported version`, func() {
			v, err := client.ValidateHypershiftVersion("4.13.0", stable)
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(BeFalse())
		})

		It(`KO: Fails to validate Hypershift version when the version is invalid or malformed`, func() {
			v, err := client.ValidateHypershiftVersion("foo.bar", stable)
			Expect(err).To(BeEquivalentTo(fmt.Errorf("version foo.bar was not found")))
			Expect(v).To(BeFalse())
		})
	})
	var _ = Context("when creating a classic cluster", func() {
		It("OK: Validates successfully a cluster with a supported version", func() {
			v, err := client.ValidateVersion("4.11.0", []string{"4.11.0"}, stable, true, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(v).To(Equal("openshift-v4.11.0"))
		})
	})
})

var _ = Describe("Validate cloud accounts", func() {

	Context("build billing accounts", func() {
		When("return the result", func() {
			It("OK: Successfully gets contracts from cloudAccounts", func() {
				mockCloudAccount := v1.NewCloudAccount().CloudAccountID("1234567").
					Contracts(v1.NewContract().StartDate(time.Now()).EndDate(time.Now().Add(2)).
						Dimensions(v1.NewContractDimension().Name("control_plane").Value("4")))
				cloudAccount, err := mockCloudAccount.Build()
				Expect(err).NotTo(HaveOccurred())
				_, isContractEnabled := ocm.GetBillingAccountContracts([]*v1.CloudAccount{cloudAccount}, "1234567")
				Expect(isContractEnabled).To(Equal(true))
			})

			It("OK: Successfully print contract details", func() {
				t, err := time.Parse(time.RFC3339, "2023-10-12T15:22:00Z")
				Expect(err).To(BeNil())
				mockContract, err := v1.NewContract().StartDate(t).
					EndDate(t).
					Dimensions(v1.NewContractDimension().Name("control_plane").Value("4"),
						v1.NewContractDimension().Name("four_vcpu_hour").Value("5")).Build()
				Expect(err).NotTo(HaveOccurred())

				expected := "\n" +
					"   +---------------------+----------------+ \n" +
					"   | Start Date          |Oct 12, 2023    | \n" +
					"   | End Date            |Oct 12, 2023    | \n" +
					"   | Number of vCPUs:    |'5'             | \n" +
					"   | Number of clusters: |'4'             | \n" +
					"   +---------------------+----------------+ \n"

				contractDisplay := ocm.GenerateContractDisplay(mockContract)

				Expect(contractDisplay).To(Equal(expected))
			})
		})
	})
})

var _ = Describe("getMachinePoolRootDisk()", func() {

	var r *rosa.Runtime
	var cmd *cobra.Command

	version := "4.10"
	isHostedCP := false
	defaultMachinePoolRootDiskSize := 12000

	BeforeEach(func() {
		r = rosa.NewRuntime()
		cmd = makeCmd()
		initFlags(cmd)

		DeferCleanup(r.Cleanup)
	})

	It("OK: isHostedCP = true", func() {

		machinePoolRootDisk, err := getMachinePoolRootDisk(r, cmd,
			version, isHostedCP, defaultMachinePoolRootDiskSize)
		Expect(err).NotTo(HaveOccurred())
		Expect(machinePoolRootDisk).To(BeNil())
	})

	It("OK: bad disk size argument", func() {
		args.machinePoolRootDiskSize = "200000000000000000000TiB"

		machinePoolRootDisk, err := getMachinePoolRootDisk(r, cmd,
			version, isHostedCP, defaultMachinePoolRootDiskSize)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected a valid machine pool root disk size value" +
			" '200000000000000000000TiB': invalid disk size: '200000000000000000000Ti'. " +
			"maximum size exceeded"))
		Expect(machinePoolRootDisk).To(BeNil())
	})

	It("OK: bad disk size argument and hcp", func() {
		args.machinePoolRootDiskSize = "200000000000000000000TiB"

		machinePoolRootDisk, err := getMachinePoolRootDisk(r, cmd,
			version, true, defaultMachinePoolRootDiskSize)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Expected a valid machine pool root disk size value" +
			" '200000000000000000000TiB': invalid disk size: '200000000000000000000Ti'. " +
			"maximum size exceeded"))
		Expect(machinePoolRootDisk).To(BeNil())
	})
})

var _ = Describe("Validations", func() {
	DescribeTable("should validate network type", func(
		in string,
		expected error,
	) {
		err := validateNetworkType(in)
		if expected == nil {
			Expect(err).To(BeNil())
		} else {
			Expect(err).To(MatchError(expected))
		}
	},
		Entry("no network type passed", "", nil),
		Entry("valid network type passed", "OpenShiftSDN", nil),
		Entry("invalid network type passed", "wrong",
			fmt.Errorf("Expected a valid network type. Valid values: %v", ocm.NetworkTypes)),
	)
})

var _ = Describe("Filtering", func() {
	r := rosa.NewRuntime()
	DescribeTable("should filter CIDR range requests", func(
		initialSubnets []ec2types.Subnet,
		machineNetwork *net.IPNet,
		serviceNetwork *net.IPNet,
		expected []ec2types.Subnet,
		expectedError string,
	) {
		out, err := filterCidrRangeSubnets(initialSubnets, machineNetwork, serviceNetwork, r)
		if expectedError == "" {
			Expect(err).To(BeNil())
			Expect(cmp.Equal(out, expected, cmpopts.IgnoreUnexported(ec2types.Subnet{}))).To(BeTrue())
		} else {
			Expect(err).To(MatchError(ContainSubstring(expectedError)))
		}
	},
		Entry(
			"no input subnets to filter",
			[]ec2types.Subnet{},           /* initialSubnets */
			mustParseCIDR("192.0.2.0/24"), /* machineNetwork */
			mustParseCIDR("142.0.0.0/16"), /* serviceNetwork */
			[]ec2types.Subnet{},           /* expected */
			"",                            /* expectedError */
		),
		Entry(
			"invalid input subnets filtered",
			[]ec2types.Subnet{ /* initialSubnets */
				{CidrBlock: aws.String("wrong"), SubnetId: aws.String("id")},
			},
			mustParseCIDR("192.0.2.0/24"), /* machineNetwork */
			mustParseCIDR("142.0.0.0/16"), /* serviceNetwork */
			nil,                           /* expected */
			"Unable to parse subnet CIDR: invalid CIDR address: wrong", /* expectedError */
		),
		Entry(
			"input subnets filtered",
			[]ec2types.Subnet{ /* initialSubnets */
				{CidrBlock: aws.String("57.0.2.0/24"), SubnetId: aws.String("id")},
				{CidrBlock: aws.String("123.244.128.0/24"), SubnetId: aws.String("id")},
				{CidrBlock: aws.String("192.0.2.0/30"), SubnetId: aws.String("id")},
				{CidrBlock: aws.String("142.6.12.0/28"), SubnetId: aws.String("id")},
			},
			mustParseCIDR("192.0.2.0/24"), /* machineNetwork */
			mustParseCIDR("142.0.0.0/16"), /* serviceNetwork */
			[]ec2types.Subnet{ /* expected */
				{CidrBlock: aws.String("192.0.2.0/30"), SubnetId: aws.String("id")},
			},
			"", /* expectedError */
		),
	)
})

var _ = Describe("hostPrefixValidator()", func() {
	It("KO: invalid format", func() {
		err := hostPrefixValidator("abc")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("strconv.Atoi: parsing \"abc\": invalid syntax"))
	})

	It("KO: short format", func() {
		err := hostPrefixValidator("123456")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Invalid Network Host Prefix /123456: Subnet length should be between 23 and 26"))
	})
})

var _ = Describe("validateBillingAccount()", func() {

	It("OK: valid billing account", func() {
		validBillingAccount := "123456789012"
		err := ocm.ValidateBillingAccount(validBillingAccount)
		Expect(err).NotTo(HaveOccurred())
	})

	It("KO: fails to validate a wrong billing account", func() {
		wrongBillingAccount := "123"
		err := ocm.ValidateBillingAccount(wrongBillingAccount)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Provided billing account number 123 is not valid. " +
			"Rerun the command with a valid billing account number"))
	})

	It("KO: fails to validate an empty billing account", func() {
		wrongBillingAccount := ""
		err := ocm.ValidateBillingAccount(wrongBillingAccount)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("A billing account number is required"))
	})

})

var _ = Describe("provideBillingAccount()", func() {

	var (
		r               *rosa.Runtime
		billingAccounts = []string{"123456789012", "987654321098"}
	)

	BeforeEach(func() {
		r = rosa.NewRuntime()
	})

	It("OK: returns the the billing account if accountID is found in billingAccounts", func() {
		accountID := "123456789012"
		billingAccount, err := provideBillingAccount(billingAccounts, accountID, r)
		Expect(billingAccount).To(Equal(billingAccount))
		Expect(err).NotTo(HaveOccurred())
	})

	It("KO: fails with an error message if accountID is not found in billingAccounts", func() {
		accountID := "000"
		billingAccount, err := provideBillingAccount(billingAccounts, accountID, r)
		Expect(billingAccount).To(Equal(""))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("A billing account is required for Hosted Control Plane clusters." +
			" To see the list of billing account options, you can use interactive mode by passing '-i'."))
	})
})

var _ = Describe("getInitialValidSubnets()", func() {

	var (
		r          *rosa.Runtime
		mockClient *mock.MockClient

		ids     = []string{"subnet-mockid-1", "subnet-mockid-2", "subnet-mockid-3", "subnet-mockid-4"}
		subnets = []ec2types.Subnet{
			{
				SubnetId:         aws.String("subnet-mockid-1"),
				AvailabilityZone: aws.String("us-east-1"),
				Tags: []ec2types.Tag{
					{
						Key:   aws.String(tags.RedHatManaged),
						Value: aws.String("true"),
					},
				},
			},
			{
				SubnetId:         aws.String("subnet-mockid-2"),
				AvailabilityZone: aws.String("us-west-1"),
			},
			{
				SubnetId:         aws.String("subnet-mockid-3"),
				AvailabilityZone: aws.String("us-west-2"),
			},
			{
				SubnetId:         aws.String("subnet-mockid-4"),
				AvailabilityZone: aws.String("us-east-2"),
			},
		}
	)

	BeforeEach(func() {
		r = rosa.NewRuntime()
		mockCtrl := gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)

		mockClient.EXPECT().ListSubnets(ids).Return(subnets, nil)
		mockClient.EXPECT().GetAvailabilityZoneType("us-east-2").Return("availability-zone", nil)
		mockClient.EXPECT().GetAvailabilityZoneType("us-west-1").Return(mock.LocalZone, nil)
		mockClient.EXPECT().GetAvailabilityZoneType("us-west-2").Return(mock.WavelengthZone, nil)
	})

	It("OK: get valid subnets", func() {
		validSubnets, err := getInitialValidSubnets(mockClient, ids, r.Reporter)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(validSubnets)).To(Equal(1))
		Expect(*validSubnets[0].SubnetId).To(Equal("subnet-mockid-4"))
	})
})

var _ = Describe("clusterHasLongNameWithoutDomainPrefix()", func() {
	DescribeTable("clusterHasLongNameWithoutDomainPrefix test cases", func(clusterName, domainPrefix string,
		expected bool) {
		actual := clusterHasLongNameWithoutDomainPrefix(clusterName, domainPrefix)
		Expect(expected).To(Equal(actual))
	},
		Entry("returns false when cluster domain prefix is given", "very-long-cluster-name-test-case",
			"domain-prefix", false),
		Entry("returns false when cluster name is shorter than 15 characters and domain prefix not given",
			"short-name", "", false),
		Entry("returns true when cluster name is longer than 15 characters and domain prefix isn't given",
			"very-long-cluster-name-test-case", "", true),
	)
})

var _ = Describe("validateUniqueIamRoleArnsForStsCluster()", func() {
	It("returns error if duplicate arns for IAM roles are found", func() {
		accountRoles := []string{"arn1", "arn2", "arn3", "arn4"}
		operatorRoles := []ocm.OperatorIAMRole{
			{RoleARN: "arn1"}, {RoleARN: "arn6"},
		}
		err := validateUniqueIamRoleArnsForStsCluster(accountRoles, operatorRoles)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf(duplicateIamRoleArnErrorMsg, "arn1")))
	})
	It("returns nil if duplicate arns for IAM roles are not found", func() {
		accountRoles := []string{"arn1", "arn2", "arn3", "arn4"}
		operatorRoles := []ocm.OperatorIAMRole{
			{RoleARN: "arn5"}, {RoleARN: "arn6"},
		}
		err := validateUniqueIamRoleArnsForStsCluster(accountRoles, operatorRoles)
		Expect(err).ToNot(HaveOccurred())
	})
})

func mustParseCIDR(s string) *net.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	Expect(err).To(BeNil())
	return ipnet
}

package interactive

import (
	"go.uber.org/mock/gomock"

	"github.com/AlecAivazis/survey/v2/core"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mock "github.com/openshift/rosa/pkg/aws"
)

var _ = Describe("Validation", func() {
	Context("MinValue", func() {
		It("Fails validation if the answer is less than the minimum", func() {
			validator := MinValue(50)
			err := validator("25")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'25' is less than the permitted minimum of '50'"))
		})

		It("Fails validation if the answer is not an integer", func() {
			validator := MinValue(50)
			err := validator("hello")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("please enter an integer value, you entered 'hello'"))
		})

		It("Fails validation if the answer is not a string", func() {
			validator := MinValue(50)
			err := validator(45)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings, got 45"))
		})

		It("Passes validation if the answer is greater than the min", func() {
			validator := MinValue(50)
			err := validator("55")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes validation if the answer is equal to the min", func() {
			validator := MinValue(50)
			err := validator("50")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("MaxValue", func() {
		It("Fails validation if the answer is greater than the maximum", func() {
			validator := MaxValue(50)
			err := validator("52")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("'52' is greater than the permitted maximum of '50'"))
		})

		It("Fails validation if the answer is not an integer", func() {
			validator := MaxValue(50)
			err := validator("hello")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("please enter an integer value, you entered 'hello'"))
		})

		It("Fails validation if the answer is not a string", func() {
			validator := MaxValue(50)
			err := validator(45)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings, got 45"))
		})

		It("Passes validation if the answer is less than the max", func() {
			validator := MaxValue(50)
			err := validator("49")
			Expect(err).NotTo(HaveOccurred())
		})

		It("Passes validation if the answer is equal to the max", func() {
			validator := MaxValue(50)
			err := validator("50")
			Expect(err).NotTo(HaveOccurred())
		})
	})
	Context("GitHub Hostname", func() {
		It("Fails validation if hostname is 'https://domain.customer.com'", func() {
			err := IsValidHostname("https://domain.customer.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"'https://domain.customer.com' hostname must be a valid DNS subdomain or IP address"),
			)
		})
		It("Fails validation if hostname is 'github.com'", func() {
			err := IsValidHostname("github.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(
				"'github.com' hostname cannot be equal to [*.]github.com"),
			)
		})
		It("Passes validation if hostname is 'domain.customer.com'", func() {
			err := IsValidHostname("domain.customer.com")
			Expect(err).NotTo(HaveOccurred())
		})
		It("Passes validation if hostname is ''", func() {
			err := IsValidHostname("")
			Expect(err).NotTo(HaveOccurred())
		})
	})

})

var _ = Describe("SubnetsValidator", func() {
	var (
		mockClient *mock.MockClient
		validator  Validator
		subnetIDs  = []string{"subnet-public-1", "subnet-private-2", "subnet-private-3"}
	)

	BeforeEach(func() {
		mockCtrl := gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
	})

	Context("When cluster is hosted", func() {
		It("Returns an error when the number of subnets is not at least two", func() {
			validator = SubnetsValidator(mockClient, false, false, true)
			answers := []core.OptionAnswer{
				{Value: "subnet-public-1 (us-west-2a)"},
			}

			err := validator(answers)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The number of subnets for a public hosted " +
				"cluster should be at least two"))
		})

		It("Returns an error when the number of public subnets is not at least one", func() {
			validator = SubnetsValidator(mockClient, false, false, true)
			answers := []core.OptionAnswer{
				{Value: "subnet-private-2 (us-west-2a)"},
				{Value: "subnet-private-3 (us-west-2b)"},
			}

			mockClient.EXPECT().GetVPCSubnets(subnetIDs[1]).Return([]ec2types.Subnet{
				{SubnetId: &subnetIDs[1]},
				{SubnetId: &subnetIDs[2]},
			}, nil)
			mockClient.EXPECT().FilterVPCsPrivateSubnets(gomock.Any()).Return([]ec2types.Subnet{
				{SubnetId: &subnetIDs[0]},
				{SubnetId: &subnetIDs[2]},
			}, nil)

			err := validator(answers)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The number of public subnets for a public hosted " +
				"cluster should be at least one"))
		})

	})

	Context("When cluster is not hosted", func() {
		It("Returns and error when the number of subnets is not correct", func() {
			validator = SubnetsValidator(mockClient, true, true, false)
			answers := []core.OptionAnswer{
				{Value: "subnet-123 (us-west-2a)"},
				{Value: "subnet-456 (us-west-2b)"},
			}

			err := validator(answers)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("The number of subnets for a 'multi-AZ'"+
				" 'private link cluster' should be '3', instead received: '%d'", len(answers)))
		})

		It("Should not return an error when the number of subnets is correct", func() {
			validator = SubnetsValidator(mockClient, true, true, false)
			answers := []core.OptionAnswer{
				{Value: "subnet-123 (us-west-2a)"},
				{Value: "subnet-456 (us-west-2b)"},
				{Value: "subnet-789 (us-west-2b)"},
			}

			err := validator(answers)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When the input is invalid", func() {
		It("Returns an error for invalid input", func() {
			err := validator("invalid input")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate a slice of string"))
		})
	})
})

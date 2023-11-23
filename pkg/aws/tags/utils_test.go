package tags

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ec2ResourceHasTag", func() {
	Context("When checking for a specific tag", func() {
		It("should return true if the tag is present", func() {
			// Test data
			tagKey := "Environment"
			tagValue := "Production"

			// Create EC2 tags
			ec2Tags := []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("Instance-1")},
				{Key: aws.String("Environment"), Value: aws.String("Production")},
				{Key: aws.String("Owner"), Value: aws.String("John Doe")},
			}

			// Call the function
			result := Ec2ResourceHasTag(ec2Tags, tagKey, tagValue)

			// Expectations
			Expect(result).To(BeTrue())
		})

		It("should return false if the tag is not present", func() {
			// Test data
			tagKey := "Environment"
			tagValue := "Staging"

			// Create EC2 tags
			ec2Tags := []types.Tag{
				{Key: aws.String("Name"), Value: aws.String("Instance-1")},
				{Key: aws.String("Environment"), Value: aws.String("Production")},
				{Key: aws.String("Owner"), Value: aws.String("John Doe")},
			}

			// Call the function
			result := Ec2ResourceHasTag(ec2Tags, tagKey, tagValue)

			// Expectations
			Expect(result).To(BeFalse())
		})
	})
})

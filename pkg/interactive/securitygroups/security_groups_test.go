package securitygroups

import (
	"fmt"
	"testing"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

func TestSecurityGroups(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Security groups Suite")
}

var _ = Describe("Security groups", func() {
	Context("Validate security group tags", func() {
		It("Return invalid for security group with red-hat-managed tag", func() {
			sg := ec2types.SecurityGroup{
				Tags: []ec2types.Tag{
					{
						Key:   awsSdk.String("red-hat-managed"),
						Value: awsSdk.String("true"),
					},
				},
			}
			isValid := isValidSecurityGroup(sg, "")
			Expect(isValid).To(Equal(false))
		})
		It("Return invalid for security group with 'default' name'", func() {
			sg := ec2types.SecurityGroup{
				GroupName: awsSdk.String("default"),
			}
			isValid := isValidSecurityGroup(sg, "")
			Expect(isValid).To(Equal(false))
		})
		It("Return valid for security group", func() {
			sg := ec2types.SecurityGroup{
				GroupName: awsSdk.String("sg-1"),
				Tags: []ec2types.Tag{
					{
						Key:   awsSdk.String("red-hat-managed"),
						Value: awsSdk.String("false"),
					},
				},
			}
			isValid := isValidSecurityGroup(sg, "")
			Expect(isValid).To(Equal(true))
		})
		It("Return invalid for security group that has a cluster tag equals owned", func() {
			infraId := "123"
			sg := ec2types.SecurityGroup{
				GroupName: awsSdk.String("sg-1"),
				Tags: []ec2types.Tag{
					{
						Key:   awsSdk.String(fmt.Sprintf("kubernetes.io/cluster/%s", infraId)),
						Value: awsSdk.String("owned"),
					},
				},
			}
			isValid := isValidSecurityGroup(sg, infraId)
			Expect(isValid).To(Equal(false))
		})
		It("Return valid for security group that has a cluster tag equals shared", func() {
			id := "123"
			sg := ec2types.SecurityGroup{
				GroupName: awsSdk.String("sg-1"),
				Tags: []ec2types.Tag{
					{
						Key:   awsSdk.String(fmt.Sprintf("kubernetes.io/cluster/%s", id)),
						Value: awsSdk.String("shared"),
					},
				},
			}
			isValid := isValidSecurityGroup(sg, id)
			Expect(isValid).To(Equal(true))
		})
	})
})

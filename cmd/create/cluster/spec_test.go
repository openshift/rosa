package cluster

import (
	"fmt"

	"github.com/google/go-cmp/cmp/cmpopts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/properties"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/openshift/rosa/pkg/test"
)

type fakeGetter struct {
	accessKeys, localAccessKeys *aws.AccessKey
}

func (f fakeGetter) GetAWSAccessKeys() (*aws.AccessKey, error) {
	if f.accessKeys != nil {
		return f.accessKeys, nil
	}
	return nil, fmt.Errorf("no AWS access keys configured on fake")
}

func (f fakeGetter) GetLocalAWSAccessKeys() (*aws.AccessKey, error) {
	if f.localAccessKeys != nil {
		return f.localAccessKeys, nil
	}
	return nil, fmt.Errorf("no AWS access keys configured on fake")
}

var _ aws.AccessKeyGetter = (*fakeGetter)(nil)

var _ = Describe("Cluster Configuration Creation", func() {
	r := rosa.NewRuntime()
	creator := &aws.Creator{
		ARN:        "test-arn",
		AccountID:  "test-account-id",
		IsSTS:      false,
		IsGovcloud: false,
	}
	DescribeTable("should create cluster configuration", func(
		in ocm.Spec,
		aws aws.AccessKeyGetter,
		expected ocm.Spec,
	) {
		out, err := clusterConfigFor(r.Reporter, in, creator, aws)
		Expect(err).To(BeNil())
		Expect(out).To(test.MatchExpected(expected, cmpopts.IgnoreUnexported(cmv1.ExternalAuthConfig{})))
	},
		Entry("no credentials required",
			ocm.Spec{
				RoleARN: "test-arn",
			},
			fakeGetter{},
			ocm.Spec{
				RoleARN: "test-arn",
			},
		),

		Entry("local credentials required",
			ocm.Spec{
				CustomProperties: map[string]string{
					properties.UseLocalCredentials: "true",
				},
			},
			fakeGetter{
				localAccessKeys: &aws.AccessKey{
					AccessKeyID:     "local-key-id",
					SecretAccessKey: "local-secret-key",
				},
			},
			ocm.Spec{
				CustomProperties: map[string]string{
					properties.UseLocalCredentials: "true",
				},
				AWSAccessKey: &aws.AccessKey{
					AccessKeyID:     "local-key-id",
					SecretAccessKey: "local-secret-key",
				},
			},
		),

		Entry("user credentials required",
			ocm.Spec{},
			fakeGetter{
				accessKeys: &aws.AccessKey{
					AccessKeyID:     "user-key-id",
					SecretAccessKey: "user-secret-key",
				},
			},
			ocm.Spec{
				AWSAccessKey: &aws.AccessKey{
					AccessKeyID:     "user-key-id",
					SecretAccessKey: "user-secret-key",
				},
			},
		),
	)
})

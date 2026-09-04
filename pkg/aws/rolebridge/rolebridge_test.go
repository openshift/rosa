package rolebridge

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/reporter"
)

func TestRolebridge(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rolebridge testing")
}

var _ = Describe("Client", func() {
	var (
		ctrl         *gomock.Controller
		mockAWS      *aws.MockClient
		testReporter reporter.Logger
		testClient   *Client
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockAWS = aws.NewMockClient(ctrl)
		testReporter = reporter.CreateReporter()
		testClient = New(mockAWS, testReporter)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	It("forwards EnsureRole to aws.Client with the held reporter, ignoring ctx", func() {
		mockAWS.EXPECT().
			EnsureRole(testReporter, "my-role", "trust-policy", "boundary", "v1", map[string]string{"k": "v"}, "/path/", true).
			Return("arn:aws:iam::123456789012:role/my-role", nil)

		roleARN, err := testClient.EnsureRole(context.Background(), "my-role", "trust-policy", "boundary",
			"v1", map[string]string{"k": "v"}, "/path/", true)
		Expect(err).ToNot(HaveOccurred())
		Expect(roleARN).To(Equal("arn:aws:iam::123456789012:role/my-role"))
	})

	It("propagates EnsureRole errors", func() {
		mockAWS.EXPECT().
			EnsureRole(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
			Return("", errors.New("access denied"))

		_, err := testClient.EnsureRole(context.Background(), "my-role", "", "", "", nil, "", false)
		Expect(err).To(MatchError("access denied"))
	})

	It("forwards AttachRolePolicy to aws.Client with the held reporter, ignoring ctx", func() {
		mockAWS.EXPECT().
			AttachRolePolicy(testReporter, "my-role", "arn:aws:iam::aws:policy/Foo").
			Return(nil)

		err := testClient.AttachRolePolicy(context.Background(), "my-role", "arn:aws:iam::aws:policy/Foo")
		Expect(err).ToNot(HaveOccurred())
	})

	It("propagates AttachRolePolicy errors", func() {
		mockAWS.EXPECT().
			AttachRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("access denied"))

		err := testClient.AttachRolePolicy(context.Background(), "my-role", "arn:aws:iam::aws:policy/Foo")
		Expect(err).To(MatchError("access denied"))
	})

	It("forwards PutRolePolicy to aws.Client, ignoring ctx", func() {
		mockAWS.EXPECT().
			PutRolePolicy("my-role", "my-policy", "{}").
			Return(nil)

		err := testClient.PutRolePolicy(context.Background(), "my-role", "my-policy", "{}")
		Expect(err).ToNot(HaveOccurred())
	})

	It("propagates PutRolePolicy errors", func() {
		mockAWS.EXPECT().
			PutRolePolicy(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(errors.New("access denied"))

		err := testClient.PutRolePolicy(context.Background(), "my-role", "my-policy", "{}")
		Expect(err).To(MatchError("access denied"))
	})
})

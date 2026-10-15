package machinepool

import (
	"context"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/pathbind"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	hfpathbind "github.com/openshift/rosa/pkg/hyperfleet/pathbind"
	"github.com/openshift/rosa/pkg/test"
)

// imageTypeEditCmd binds --image-type the way the generated registration does
// and marks it changed, including for the empty value a user can pass.
func imageTypeEditCmd(input *hfpathbind.NodePoolUpdateInput, value string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringVar(&input.ImageType, "image-type", "", "")
	Expect(cmd.Flags().Set("image-type", value)).To(Succeed())
	return cmd
}

// nodePoolWithImageType builds a node pool the way the API reports one.
func nodePoolWithImageType(imageType hypershiftv1beta1.ImageType) *v1alpha1.NodePool {
	return &v1alpha1.NodePool{
		ObjectMeta: metav1.ObjectMeta{Name: "my-cluster.workers", UID: "pool-uid"},
		Spec: v1alpha1.NodePoolSpec{NodePool: v1alpha1.NodePoolSpecPassthrough{
			Platform: v1alpha1.NodePoolPlatform{AWS: &v1alpha1.AWSNodePoolPlatform{
				InstanceType: "m5.xlarge", ImageType: imageType,
			}},
		}},
	}
}

var _ = Describe("HyperFleet machine pool image type editing", func() {
	// editImageType runs the handler the way runHyperfleetEdit does, against a
	// node pool whose image type is currently Linux.
	editImageType := func(value string) (*v1alpha1.NodePool, error) {
		t := test.NewTestRuntime()
		ctrl := gomock.NewController(GinkgoT())
		hf, _, pools := newEditMPMocks(ctrl)
		t.RosaRuntime.HyperFleetClient = hf
		pools.EXPECT().Get(gomock.Any(), "my-cluster.workers", gomock.Any()).
			Return(nodePoolWithImageType(hypershiftv1beta1.ImageTypeLinux), nil).AnyTimes()

		input := &hfpathbind.NodePoolUpdateInput{}
		cmd := imageTypeEditCmd(input, value)
		handler := &hyperfleetNodePoolUpdate{
			clusterKey: "my-cluster", nodePoolKey: "workers", nodePoolName: "my-cluster.workers",
			userOptions: &EditMachinepoolUserOptions{}, cmd: cmd,
		}
		if err := handler.PreRequest(context.Background(), t.RosaRuntime, input); err != nil {
			return nil, err
		}
		obj := &v1alpha1.NodePool{}
		if err := pathbind.Expand(context.Background(), *input, obj); err != nil {
			return nil, err
		}
		err := handler.PostExpand(context.Background(), t.RosaRuntime, input, obj)
		return obj, err
	}

	It("switches a node pool to Windows", func() {
		updated, err := editImageType("Windows")
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.NodePool.Platform.AWS.ImageType).To(Equal(hypershiftv1beta1.ImageTypeWindows))
	})

	It("accepts the OCM spelling of the default", func() {
		updated, err := editImageType("Default")
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.NodePool.Platform.AWS.ImageType).To(Equal(hypershiftv1beta1.ImageTypeLinux))
	})

	It("keeps the rest of the node pool as the API reported it", func() {
		updated, err := editImageType("Windows")
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.Spec.NodePool.Platform.AWS.InstanceType).To(Equal("m5.xlarge"))
		Expect(updated.Name).To(Equal("my-cluster.workers"))
	})

	It("rejects an unknown image type", func() {
		_, err := editImageType("Solaris")
		Expect(err).To(MatchError(ContainSubstring("expected a valid image type for the machine pool: 'Solaris'")))
	})

	It("rejects clearing the image type, which the API cannot express", func() {
		input := &hfpathbind.NodePoolUpdateInput{}
		cmd := imageTypeEditCmd(input, "")
		handler := &hyperfleetNodePoolUpdate{userOptions: &EditMachinepoolUserOptions{}, cmd: cmd}
		err := handler.PreRequest(context.Background(), test.NewTestRuntime().RosaRuntime, input)
		Expect(err).To(MatchError("--image-type requires a value: Linux or Windows"))
	})

	It("counts as a supported flag on its own", func() {
		input := &hfpathbind.NodePoolUpdateInput{}
		cmd := imageTypeEditCmd(input, "Windows")
		handler := &hyperfleetNodePoolUpdate{userOptions: &EditMachinepoolUserOptions{}, cmd: cmd}
		Expect(handler.PreRequest(context.Background(), test.NewTestRuntime().RosaRuntime, input)).To(Succeed())
	})
})

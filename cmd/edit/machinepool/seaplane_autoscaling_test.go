// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package machinepool

import (
	"context"

	"go.uber.org/mock/gomock"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-seaplane-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	"github.com/spf13/cobra"

	rosaruntime "github.com/openshift/rosa/pkg/rosa"
	seaplanepathbind "github.com/openshift/rosa/pkg/seaplane/pathbind"
)

func autoscalingEditCmd(options *EditMachinepoolUserOptions, flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().IntVar(&options.replicas, "replicas", 0, "")
	cmd.Flags().BoolVar(&options.autoscalingEnabled, "enable-autoscaling", false, "")
	cmd.Flags().IntVar(&options.minReplicas, "min-replicas", 0, "")
	cmd.Flags().IntVar(&options.maxReplicas, "max-replicas", 0, "")
	for flag, value := range flags {
		Expect(cmd.Flags().Set(flag, value)).To(Succeed())
	}
	return cmd
}

var _ = Describe("Seaplane autoscaling edits", func() {
	DescribeTable("merges partial autoscaling bounds", func(flags map[string]string, expected *hypershiftv1beta1.NodePoolAutoScaling) {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, pools := newEditMPMocks(ctrl)
		runtime := rosaruntime.NewRuntime()
		runtime.SeaplaneClient = seaplaneClient

		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.workers", UID: types.UID("pool-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					AutoScaling: &hypershiftv1beta1.NodePoolAutoScaling{
						Min: ptr.To(int32(1)),
						Max: 3,
					},
				},
			},
		}
		pools.EXPECT().Get(gomock.Any(), "cluster1.workers", gomock.Any()).Return(current, nil)
		options := &EditMachinepoolUserOptions{}
		handler := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "workers",
			nodePoolName: "cluster1.workers",
			userOptions:  options,
			cmd:          autoscalingEditCmd(options, flags),
		}
		input := &seaplanepathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), runtime, input)).To(Succeed())
		updated := &v1alpha1.NodePool{}
		Expect(handler.PostExpand(context.Background(), runtime, input, updated)).To(Succeed())
		Expect(updated.Spec.NodePool.AutoScaling).To(Equal(expected))
		Expect(updated.Spec.NodePool.Replicas).To(BeNil())
	},
		Entry("updates only max replicas", map[string]string{"max-replicas": "5"},
			&hypershiftv1beta1.NodePoolAutoScaling{Min: ptr.To(int32(1)), Max: 5}),
		Entry("updates only min replicas", map[string]string{"min-replicas": "2"},
			&hypershiftv1beta1.NodePoolAutoScaling{Min: ptr.To(int32(2)), Max: 3}),
		Entry("preserves existing bounds when autoscaling is enabled explicitly", map[string]string{"enable-autoscaling": "true"},
			&hypershiftv1beta1.NodePoolAutoScaling{Min: ptr.To(int32(1)), Max: 3}),
	)

	It("defaults an omitted bound from fixed replicas when enabling autoscaling", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, pools := newEditMPMocks(ctrl)
		runtime := rosaruntime.NewRuntime()
		runtime.SeaplaneClient = seaplaneClient

		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.workers", UID: types.UID("pool-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
				},
			},
		}
		pools.EXPECT().Get(gomock.Any(), "cluster1.workers", gomock.Any()).Return(current, nil)

		options := &EditMachinepoolUserOptions{}
		handler := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "workers",
			nodePoolName: "cluster1.workers",
			userOptions:  options,
			cmd: autoscalingEditCmd(options, map[string]string{
				"enable-autoscaling": "true",
				"max-replicas":       "5",
			}),
		}
		input := &seaplanepathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), runtime, input)).To(Succeed())
		updated := &v1alpha1.NodePool{}
		Expect(handler.PostExpand(context.Background(), runtime, input, updated)).To(Succeed())
		Expect(updated.Spec.NodePool.AutoScaling).To(Equal(&hypershiftv1beta1.NodePoolAutoScaling{
			Min: ptr.To(int32(3)),
			Max: 5,
		}))
		Expect(updated.Spec.NodePool.Replicas).To(BeNil())
	})

	It("rejects changing an autoscaling bound when autoscaling is not enabled", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, pools := newEditMPMocks(ctrl)
		runtime := rosaruntime.NewRuntime()
		runtime.SeaplaneClient = seaplaneClient

		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.workers", UID: types.UID("pool-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					Replicas: ptr.To(int32(3)),
				},
			},
		}
		pools.EXPECT().Get(gomock.Any(), "cluster1.workers", gomock.Any()).Return(current, nil)

		options := &EditMachinepoolUserOptions{}
		handler := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "workers",
			nodePoolName: "cluster1.workers",
			userOptions:  options,
			cmd:          autoscalingEditCmd(options, map[string]string{"max-replicas": "5"}),
		}
		input := &seaplanepathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), runtime, input)).To(Succeed())
		updated := &v1alpha1.NodePool{}
		err := handler.PostExpand(context.Background(), runtime, input, updated)
		Expect(err).To(MatchError("autoscaling is not enabled on node pool 'workers'. can't set min or max replicas"))
	})

	It("validates the merged autoscaling range", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, pools := newEditMPMocks(ctrl)
		runtime := rosaruntime.NewRuntime()
		runtime.SeaplaneClient = seaplaneClient

		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.workers", UID: types.UID("pool-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					AutoScaling: &hypershiftv1beta1.NodePoolAutoScaling{
						Min: ptr.To(int32(5)),
						Max: 6,
					},
				},
			},
		}
		pools.EXPECT().Get(gomock.Any(), "cluster1.workers", gomock.Any()).Return(current, nil)

		options := &EditMachinepoolUserOptions{}
		handler := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "workers",
			nodePoolName: "cluster1.workers",
			userOptions:  options,
			cmd:          autoscalingEditCmd(options, map[string]string{"max-replicas": "4"}),
		}
		input := &seaplanepathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), runtime, input)).To(Succeed())
		updated := &v1alpha1.NodePool{}
		err := handler.PostExpand(context.Background(), runtime, input, updated)
		Expect(err).To(MatchError("max-replicas must be greater than or equal to min-replicas"))
	})

	It("rejects autoscaling bounds when autoscaling is explicitly disabled", func() {
		ctrl := gomock.NewController(GinkgoT())
		seaplaneClient, _, pools := newEditMPMocks(ctrl)
		runtime := rosaruntime.NewRuntime()
		runtime.SeaplaneClient = seaplaneClient

		current := &v1alpha1.NodePool{
			ObjectMeta: metav1.ObjectMeta{Name: "cluster1.workers", UID: types.UID("pool-uid")},
			Spec: v1alpha1.NodePoolSpec{
				NodePool: v1alpha1.NodePoolSpecPassthrough{
					AutoScaling: &hypershiftv1beta1.NodePoolAutoScaling{
						Min: ptr.To(int32(1)),
						Max: 3,
					},
				},
			},
		}
		pools.EXPECT().Get(gomock.Any(), "cluster1.workers", gomock.Any()).Return(current, nil)

		options := &EditMachinepoolUserOptions{}
		handler := &seaplaneNodePoolUpdate{
			clusterKey:   "cluster1",
			nodePoolKey:  "workers",
			nodePoolName: "cluster1.workers",
			userOptions:  options,
			cmd: autoscalingEditCmd(options, map[string]string{
				"enable-autoscaling": "false",
				"max-replicas":       "5",
			}),
		}
		input := &seaplanepathbind.NodePoolUpdateInput{}
		Expect(handler.PreRequest(context.Background(), runtime, input)).To(Succeed())
		updated := &v1alpha1.NodePool{}
		err := handler.PostExpand(context.Background(), runtime, input, updated)
		Expect(err).To(MatchError("autoscaling is not enabled on node pool 'workers'. can't set min or max replicas"))
	})
})

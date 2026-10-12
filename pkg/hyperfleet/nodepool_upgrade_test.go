package hyperfleet

import (
	"k8s.io/apimachinery/pkg/util/intstr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

var _ = Describe("NodePoolUpgradeSettings", func() {
	nodePoolWithManagement := func(management hypershiftv1beta1.NodePoolManagement) *v1alpha1.NodePool {
		return &v1alpha1.NodePool{Spec: v1alpha1.NodePoolSpec{
			NodePool: v1alpha1.NodePoolSpecPassthrough{Management: management},
		}}
	}

	It("reports the rendered absolute values", func() {
		maxSurge := intstr.FromInt32(2)
		maxUnavailable := intstr.FromInt32(1)
		upgradeType, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
			Replace: &hypershiftv1beta1.ReplaceUpgrade{
				Strategy: hypershiftv1beta1.UpgradeStrategyRollingUpdate,
				RollingUpdate: &hypershiftv1beta1.RollingUpdate{
					MaxSurge: &maxSurge, MaxUnavailable: &maxUnavailable,
				},
			},
		}))
		Expect(upgradeType).To(Equal("Replace"))
		Expect(upgrade).To(Equal(&NodePoolUpgrade{MaxSurge: "2", MaxUnavailable: "1"}))
	})

	It("reports percentage values as the API rendered them", func() {
		maxSurge := intstr.FromString("20%")
		maxUnavailable := intstr.FromString("10%")
		_, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			Replace: &hypershiftv1beta1.ReplaceUpgrade{RollingUpdate: &hypershiftv1beta1.RollingUpdate{
				MaxSurge: &maxSurge, MaxUnavailable: &maxUnavailable,
			}},
		}))
		Expect(upgrade).To(Equal(&NodePoolUpgrade{MaxSurge: "20%", MaxUnavailable: "10%"}))
	})

	It("reports an unset value as empty rather than as a default", func() {
		maxSurge := intstr.FromInt32(1)
		_, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			Replace: &hypershiftv1beta1.ReplaceUpgrade{RollingUpdate: &hypershiftv1beta1.RollingUpdate{
				MaxSurge: &maxSurge,
			}},
		}))
		Expect(upgrade).To(Equal(&NodePoolUpgrade{MaxSurge: "1", MaxUnavailable: ""}))
	})

	It("returns no upgrade when the API has not rendered a rolling update", func() {
		upgradeType, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			UpgradeType: hypershiftv1beta1.UpgradeTypeInPlace,
		}))
		Expect(upgradeType).To(Equal("InPlace"))
		Expect(upgrade).To(BeNil())
	})

	It("returns no upgrade for a node pool without management settings", func() {
		upgradeType, upgrade := NodePoolUpgradeSettings(&v1alpha1.NodePool{})
		Expect(upgradeType).To(BeEmpty())
		Expect(upgrade).To(BeNil())
	})
})

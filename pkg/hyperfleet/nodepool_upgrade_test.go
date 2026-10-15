package hyperfleet

import (
	"encoding/json"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1alpha1 "github.com/openshift-online/rosa-hyperfleet-api/api/v1alpha1/public"
	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
)

var _ = Describe("EncodeUpgradeValue", func() {
	// The generated input decodes these as JSON into an int-or-string field,
	// so a percentage has to arrive quoted.
	DescribeTable("renders the JSON form the int-or-string field expects",
		func(value, expected string) {
			encoded := EncodeUpgradeValue(value)
			Expect(encoded).To(Equal(expected))
			if encoded == "" {
				return
			}
			decoded := &intstr.IntOrString{}
			Expect(json.Unmarshal([]byte(encoded), decoded)).To(Succeed())
			Expect(decoded.String()).To(Equal(strings.TrimSpace(value)))
		},
		Entry("absolute count", "2", "2"),
		Entry("zero", "0", "0"),
		Entry("percentage", "20%", `"20%"`),
		Entry("zero percentage", "0%", `"0%"`),
		Entry("surrounding space", " 2 ", "2"),
		Entry("unset", "", ""),
	)
})

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
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxSurge: "2", MaxUnavailable: "1"}))
	})

	It("reports percentage values as the API rendered them", func() {
		maxSurge := intstr.FromString("20%")
		maxUnavailable := intstr.FromString("10%")
		_, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			Replace: &hypershiftv1beta1.ReplaceUpgrade{RollingUpdate: &hypershiftv1beta1.RollingUpdate{
				MaxSurge: &maxSurge, MaxUnavailable: &maxUnavailable,
			}},
		}))
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxSurge: "20%", MaxUnavailable: "10%"}))
	})

	It("falls back to the API default for a half nobody set", func() {
		maxSurge := intstr.FromInt32(1)
		_, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			Replace: &hypershiftv1beta1.ReplaceUpgrade{RollingUpdate: &hypershiftv1beta1.RollingUpdate{
				MaxSurge: &maxSurge,
			}},
		}))
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxSurge: "1", MaxUnavailable: "0"}))
	})

	It("leaves max surge out for an InPlace node pool, which never surges", func() {
		upgradeType, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			UpgradeType: hypershiftv1beta1.UpgradeTypeInPlace,
		}))
		Expect(upgradeType).To(Equal("InPlace"))
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxUnavailable: "0"}))
	})

	It("reports the API defaults for a node pool the customer never configured", func() {
		upgradeType, upgrade := NodePoolUpgradeSettings(&v1alpha1.NodePool{})
		Expect(upgradeType).To(BeEmpty())
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxSurge: "1", MaxUnavailable: "0"}))
	})

	It("prefers the customer values on spec over the rendered management", func() {
		np := nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
			Replace: &hypershiftv1beta1.ReplaceUpgrade{RollingUpdate: &hypershiftv1beta1.RollingUpdate{
				MaxSurge: ptr.To(intstr.FromInt32(1)), MaxUnavailable: ptr.To(intstr.FromInt32(0)),
			}},
		})
		np.Spec.MaxSurge = ptr.To(intstr.FromString("20%"))
		np.Spec.MaxUnavailable = ptr.To(intstr.FromInt32(2))

		_, upgrade := NodePoolUpgradeSettings(np)
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxSurge: "20%", MaxUnavailable: "2"}))
	})

	It("falls back to the rendered value for the half the customer did not set", func() {
		np := nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			UpgradeType: hypershiftv1beta1.UpgradeTypeReplace,
			Replace: &hypershiftv1beta1.ReplaceUpgrade{RollingUpdate: &hypershiftv1beta1.RollingUpdate{
				MaxSurge: ptr.To(intstr.FromInt32(1)), MaxUnavailable: ptr.To(intstr.FromInt32(0)),
			}},
		})
		np.Spec.MaxUnavailable = ptr.To(intstr.FromInt32(2))

		_, upgrade := NodePoolUpgradeSettings(np)
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxSurge: "1", MaxUnavailable: "2"}))
	})

	It("reports the customer values before the API renders management", func() {
		np := &v1alpha1.NodePool{Spec: v1alpha1.NodePoolSpec{
			MaxSurge: ptr.To(intstr.FromInt32(3)),
		}}
		upgradeType, upgrade := NodePoolUpgradeSettings(np)
		Expect(upgradeType).To(BeEmpty())
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxSurge: "3", MaxUnavailable: "0"}))
	})

	It("reports the in-place maxUnavailable for an InPlace node pool", func() {
		_, upgrade := NodePoolUpgradeSettings(nodePoolWithManagement(hypershiftv1beta1.NodePoolManagement{
			UpgradeType: hypershiftv1beta1.UpgradeTypeInPlace,
			InPlace:     &hypershiftv1beta1.InPlaceUpgrade{MaxUnavailable: ptr.To(intstr.FromInt32(1))},
		}))
		Expect(upgrade).To(Equal(NodePoolUpgrade{MaxUnavailable: "1"}))
	})
})

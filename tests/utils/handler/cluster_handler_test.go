package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/tests/utils/constants"
)

var _ = Describe("classifyClusterState", func() {
	It("recognizes ready state", func() {
		Expect(classifyClusterState(constants.Ready)).To(Equal(clusterStateReady))
	})

	It("recognizes uninstalling as terminal", func() {
		Expect(classifyClusterState(constants.Uninstalling)).To(Equal(clusterStateTerminal))
	})

	It("recognizes error as terminal", func() {
		Expect(classifyClusterState(constants.Error)).To(Equal(clusterStateTerminal))
	})

	It("recognizes waiting state", func() {
		Expect(classifyClusterState(constants.Waiting)).To(Equal(clusterStateWaiting))
	})

	It("recognizes installing as transient", func() {
		Expect(classifyClusterState(constants.Installing)).To(Equal(clusterStateTransient))
	})

	It("recognizes pending as transient", func() {
		Expect(classifyClusterState(constants.Pending)).To(Equal(clusterStateTransient))
	})

	It("recognizes validating as transient", func() {
		Expect(classifyClusterState(constants.Validating)).To(Equal(clusterStateTransient))
	})

	It("returns unknown for unrecognized states", func() {
		Expect(classifyClusterState("hibernating")).To(Equal(clusterStateUnknown))
	})

	It("returns unknown for empty state", func() {
		Expect(classifyClusterState("")).To(Equal(clusterStateUnknown))
	})
})

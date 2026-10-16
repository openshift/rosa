package rosacli

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nodepool lookup", func() {
	It("resolves HyperFleet nodepools by name or UID", func() {
		pool := &NodePool{ID: "pool-uid", Name: "workers"}
		list := NodePoolList{NodePools: []*NodePool{pool}}
		Expect(list.Nodepool("workers")).To(BeIdenticalTo(pool))
		Expect(list.Nodepool("pool-uid")).To(BeIdenticalTo(pool))
		Expect(list.Nodepool("missing")).To(BeNil())
	})
	It("retains OCM lookup by ID without a name column", func() {
		pool := &NodePool{ID: "workers"}
		list := NodePoolList{NodePools: []*NodePool{pool}}
		Expect(list.Nodepool("workers")).To(BeIdenticalTo(pool))
		Expect(list.Nodepool("")).To(BeNil())
	})
})

var _ = Describe("Nodepool replica readiness", func() {
	It("retries transient node pool describe errors", func() {
		binDir := GinkgoT().TempDir()
		script := filepath.Join(binDir, "rosa")
		scriptContent := `#!/bin/sh
echo "the server is currently unable to handle the request" >&2
exit 1
`
		Expect(os.WriteFile(script, []byte(scriptContent), 0700)).To(Succeed())

		originalPath := os.Getenv("PATH")
		os.Setenv("PATH", binDir+":"+originalPath)
		DeferCleanup(func() {
			os.Setenv("PATH", originalPath)
		})

		service := NewMachinePoolService(NewClient())
		err := service.WaitForNodePoolReplicasReady(
			"cluster-id", "node-pool", false, time.Millisecond, 5*time.Millisecond)

		Expect(err).To(MatchError(ContainSubstring("context deadline exceeded")))
		Expect(err.Error()).NotTo(ContainSubstring("the server is currently unable to handle the request"))
	})

	It("identifies transient node pool polling errors", func() {
		Expect(isTransientNodePoolPollError(
			errors.New("exit status 1: the server is currently unable to handle the request"),
		)).To(BeTrue())
		Expect(isTransientNodePoolPollError(errors.New("exit status 1: cluster not found"))).To(BeFalse())
	})
})

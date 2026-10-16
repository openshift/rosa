// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosacli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReflectNodePoolDescription", func() {
	It("parses a node pool description whose status Message is multi-line bullet text", func() {
		output := `
ID:                                    workers-0
Cluster ID:                            2t63qi7a1shbrbh9etlq1kjcia07ib9t
Autoscaling:                           No
Desired replicas:                      1
Current replicas:                      0
Instance type:                         m7i.xlarge
Disk Size:                             300 GiB
Version:                               4.22.15
EC2 Metadata Http Tokens:              required
Autorepair:                            Yes
Management upgrade:
 - Type:                               Replace
 - Max surge:                          1
 - Max unavailable:                    0
Message:                               * Machine rosa-hcppl-f3-q-workers-0-5tg6x-l8pbz:
  * NodeHealthy: Waiting for a Node with spec.providerID aws:///us-west-2a/i-00dfda71eb9b2bc65 to exist
`
		m := &machinepoolService{ResourcesService: ResourcesService{client: &Client{Parser: NewParser()}}}
		var buf bytes.Buffer
		buf.WriteString(output)

		npd, err := m.ReflectNodePoolDescription(buf)

		Expect(err).ToNot(HaveOccurred())
		Expect(npd.ID).To(Equal("workers-0"))
		Expect(npd.InstanceType).To(Equal("m7i.xlarge"))
		Expect(npd.DiskSize).To(Equal("300 GiB"))
		Expect(npd.Message).To(ContainSubstring("Machine rosa-hcppl-f3-q-workers-0-5tg6x-l8pbz"))
		Expect(npd.Message).To(ContainSubstring("NodeHealthy: Waiting for a Node"))
	})

	It("preserves Message content that looks like CLI log noise", func() {
		output := `
ID:                                    workers-0
Cluster ID:                            2t63qi7a1shbrbh9etlq1kjcia07ib9t
Autoscaling:                           No
Instance type:                         m7i.xlarge
Message:                               * Machine foo:
  * Error: something went Wrong with multiple words here
`
		m := &machinepoolService{ResourcesService: ResourcesService{client: &Client{Parser: NewParser()}}}
		var buf bytes.Buffer
		buf.WriteString(output)

		npd, err := m.ReflectNodePoolDescription(buf)

		Expect(err).ToNot(HaveOccurred())
		Expect(npd.Message).To(ContainSubstring("Error: something went Wrong with multiple words here"))
	})

	It("does not corrupt a Message line containing multiple colons", func() {
		output := `
ID:                                    workers-0
Cluster ID:                            2t63qi7a1shbrbh9etlq1kjcia07ib9t
Autoscaling:                           No
Instance type:                         m7i.xlarge
Message:                               * foo: bar: baz
`
		m := &machinepoolService{ResourcesService: ResourcesService{client: &Client{Parser: NewParser()}}}
		var buf bytes.Buffer
		buf.WriteString(output)

		npd, err := m.ReflectNodePoolDescription(buf)

		Expect(err).ToNot(HaveOccurred())
		Expect(npd.Message).To(Equal("* foo: bar: baz"))
	})

	It("leaves a healthy node pool's empty Message untouched", func() {
		output := `
ID:                                    workers-0
Cluster ID:                            2t63qi7a1shbrbh9etlq1kjcia07ib9t
Autoscaling:                           No
Instance type:                         m7i.xlarge
Message:
`
		m := &machinepoolService{ResourcesService: ResourcesService{client: &Client{Parser: NewParser()}}}
		var buf bytes.Buffer
		buf.WriteString(output)

		npd, err := m.ReflectNodePoolDescription(buf)

		Expect(err).ToNot(HaveOccurred())
		Expect(npd.Message).To(BeEmpty())
	})

	It("parses Platform API node pool conditions with multi-line bullet text", func() {
		output := `
ID:                                    workers-0
Cluster ID:                            2t63qi7a1shbrbh9etlq1kjcia07ib9t
State:                                 Provisioning
Autoscaling:                           No
Desired replicas:                      1
Current replicas:
Instance type:                         m7i.xlarge
Conditions:
 - Synced:    True   AllSynced: 1/1 desires applied successfully
 - Ready:     False  NotReady: * Machine workers-0-abc:
  * NodeHealthy: Waiting for a Node with spec.providerID aws:///us-east-1a/i-0123456789abcdef0 to exist
`
		m := &machinepoolService{ResourcesService: ResourcesService{client: &Client{Parser: NewParser()}}}
		var buf bytes.Buffer
		buf.WriteString(output)

		npd, err := m.ReflectNodePoolDescription(buf)

		Expect(err).ToNot(HaveOccurred())
		Expect(npd.ID).To(Equal("workers-0"))
		Expect(npd.State).To(Equal("Provisioning"))
		Expect(npd.CurrentReplicas).To(BeEmpty())
	})
})

var _ = Describe("Nodepool lookup", func() {
	It("resolves Seaplane nodepools by name or UID", func() {
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

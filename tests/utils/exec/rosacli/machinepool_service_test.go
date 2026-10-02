// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosacli

import (
	"bytes"

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
})

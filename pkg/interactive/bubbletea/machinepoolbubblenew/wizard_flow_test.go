/*
Copyright (c) 2021 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machinepoolbubblenew

import (
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	tea "github.com/charmbracelet/bubbletea"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
	"github.com/openshift/rosa/pkg/machinepooldemo"
)

var _ = Describe("Machine pool UX wizard flow", func() {
	It("starts on machine pool name with progress header", func() {
		m := newTestWizard()
		Expect(m.step).To(Equal(stepName))
		Expect(m.View()).To(ContainSubstring("Step 1/24"))
		Expect(m.View()).To(ContainSubstring("Machine pool name"))
	})

	It("rejects an empty machine pool name", func() {
		m := newTestWizard()
		m = pressEnter(m)
		Expect(m.step).To(Equal(stepName))
		Expect(m.errMsg).To(Equal("machine pool name is required"))
		Expect(m.View()).To(ContainSubstring("machine pool name is required"))
	})

	It("keeps completed answers visible while advancing", func() {
		m := newTestWizard()
		m = setText(m, "worker-pool")
		m = pressEnter(m)
		Expect(m.step).To(Equal(stepImageType))
		Expect(m.View()).To(ContainSubstring("Machine pool name"))
		Expect(m.View()).To(ContainSubstring("worker-pool"))
		Expect(m.completed).To(HaveLen(1))
	})

	It("requires golden-path subnet and autoscaling choices", func() {
		m := advanceToSubnetSelection(newTestWizard())
		m = pressUp(m)
		m = pressEnter(m)
		Expect(m.step).To(Equal(stepSelectSubnet))
		Expect(m.errMsg).To(Equal(machinepooldemo.MsgGoldenPathSubnet))

		m = pressDown(m)
		m = pressEnter(m)
		m = advanceToAutoscaling(m)
		m = pressEnter(m)
		Expect(m.step).To(Equal(stepAutoscaling))
		Expect(m.errMsg).To(Equal(machinepooldemo.MsgGoldenPathAutoscaling))
	})

	It("toggles checkbox list selections with space", func() {
		m := advanceToSecurityGroups(newTestWizard())
		Expect(m.View()).To(ContainSubstring("○"))

		m = pressSpace(m)
		Expect(m.checkbox.selected).To(HaveLen(1))
		Expect(m.View()).To(ContainSubstring("✓"))

		m = pressEnter(m)
		Expect(m.step).To(Equal(stepTags))
		Expect(m.result.SecurityGroupIDs).To(HaveLen(1))
	})

	It("allows only one kubelet config selection at a time", func() {
		m := advanceToKubeletConfig(newTestWizard())
		names := machinepooldemo.KubeletConfigNames()
		m = pressSpace(m)
		Expect(m.checkbox.selected).To(HaveLen(1))
		m = pressDown(m)
		m = pressSpace(m)
		Expect(m.checkbox.selected).To(HaveLen(1))
		_, selected := m.checkbox.selected[names[1]]
		Expect(selected).To(BeTrue())
		m = pressEnter(m)
		Expect(m.step).To(Equal(stepHTTPTokens))
		Expect(m.result.KubeletConfigs).To(Equal([]string{names[1]}))
	})

	It("walks the golden path and records the expected result", func() {
		m := walkGoldenPath(newTestWizard())
		Expect(m.done).To(BeTrue())
		Expect(m.completed).To(HaveLen(24))
		Expect(m.result.Name).To(Equal("worker-pool"))
		Expect(m.result.Version).To(Equal(machinepooldemo.DemoClusterVersion))
		Expect(m.result.Autoscaling).To(BeTrue())
		Expect(m.result.MinReplicas).To(Equal(2))
		Expect(m.result.MaxReplicas).To(Equal(4))
		Expect(m.result.Labels).To(Equal("abc=123"))
		Expect(m.result.InstanceType).To(Equal(machinepooldemo.InstanceTypes()[0]))
		Expect(m.result.Autorepair).To(BeTrue())
		Expect(m.result.RootDiskSize).To(Equal(machinepooldemo.DefaultDiskSize))
		Expect(m.result.MaxSurge).To(Equal(machinepooldemo.DefaultMaxSurge))
		Expect(m.result.MaxUnavailable).To(Equal(machinepooldemo.DefaultMaxUnavail))
		Expect(m.result.HTTPTokens).To(Equal(string(cmv1.Ec2MetadataHttpTokensOptional)))
		Expect(m.result.Subnet).To(Equal(aws.ParseOption(machinepooldemo.SubnetOptions()[0])))
	})
})

func newTestWizard() wizardModel {
	return newWizardModel()
}

func pressEnter(m wizardModel) wizardModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(wizardModel)
}

func pressDown(m wizardModel) wizardModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	return next.(wizardModel)
}

func pressUp(m wizardModel) wizardModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	return next.(wizardModel)
}

func pressSpace(m wizardModel) wizardModel {
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	return next.(wizardModel)
}

func setText(m wizardModel, value string) wizardModel {
	m.text.SetValue(value)
	return m
}

func advanceToSubnetSelection(m wizardModel) wizardModel {
	m = setText(m, "worker-pool")
	m = pressEnter(m)
	m = pressEnter(m) // image type default
	m = pressEnter(m) // version
	return m
}

func advanceToAutoscaling(m wizardModel) wizardModel {
	m = advanceToSubnetSelection(m)
	m = pressEnter(m) // subnet no
	m = pressEnter(m) // az
	m = pressEnter(m) // subnet id
	return m
}

func advanceToSecurityGroups(m wizardModel) wizardModel {
	m = advanceToAutoscaling(m)
	m = pressUp(m)
	m = pressEnter(m) // autoscaling yes
	m = pressEnter(m) // min
	m = pressEnter(m) // max
	m = setText(m, "abc=123")
	m = pressEnter(m) // labels
	m = pressEnter(m) // taints
	return m
}

func advanceToKubeletConfig(m wizardModel) wizardModel {
	m = advanceToSecurityGroups(m)
	m = pressEnter(m) // security groups
	m = pressEnter(m) // tags
	m = pressEnter(m) // instance type
	m = pressEnter(m) // autorepair
	m = pressEnter(m) // tuning configs
	m = pressEnter(m) // capacity reservation id
	m = pressEnter(m) // capacity preference skip
	return m
}

func walkGoldenPath(m wizardModel) wizardModel {
	m = advanceToSecurityGroups(m)
	for i := 0; i < 13; i++ {
		m = pressEnter(m)
	}
	return m
}

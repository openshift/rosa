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

package machinepoolbubble

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/openshift/rosa/pkg/aws"
	mpHelpers "github.com/openshift/rosa/pkg/helper/machinepools"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/consts"
	"github.com/openshift/rosa/pkg/machinepool"
	"github.com/openshift/rosa/pkg/machinepooldemo"
	"github.com/openshift/rosa/pkg/ocm"
)

type wizardStep int

const (
	stepName wizardStep = iota
	stepImageType
	stepVersion
	stepSelectSubnet
	stepAvailabilityZone
	stepSubnetID
	stepAutoscaling
	stepMinReplicas
	stepMaxReplicas
	stepLabels
	stepTaints
	stepSecurityGroups
	stepTags
	stepInstanceType
	stepAutorepair
	stepTuningConfigs
	stepCapacityReservationID
	stepCapacityPreference
	stepKubeletConfig
	stepHTTPTokens
	stepRootDiskSize
	stepNodeDrainGracePeriod
	stepMaxSurge
	stepMaxUnavailable
)

type optionItem struct {
	value string
}

func (i optionItem) Title() string       { return i.value }
func (i optionItem) Description() string { return "" }
func (i optionItem) FilterValue() string { return i.value }

type boolItem struct {
	value bool
	label string
}

func (i boolItem) Title() string       { return i.label }
func (i boolItem) Description() string { return "" }
func (i boolItem) FilterValue() string { return i.label }

type completedAnswer struct {
	label string
	value string
}

type wizardModel struct {
	step              wizardStep
	result            machinepooldemo.Result
	text              textinput.Model
	list              list.Model
	multi             multiSelectModel
	completed         []completedAnswer
	errMsg            string
	done              bool
	aborted           bool
	replicaValidation *machinepool.ReplicaSizeValidation
}

// RunWizard collects machine pool settings using Bubble Tea prompts.
func RunWizard() (machinepooldemo.Result, error) {
	if !isTerminal(os.Stdout) {
		return machinepooldemo.Result{}, fmt.Errorf("machine pool bubble demo requires an interactive terminal")
	}

	model := newWizardModel()
	final, err := tea.NewProgram(&model, tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		return machinepooldemo.Result{}, err
	}
	resultModel, ok := wizardResult(final)
	if !ok {
		return machinepooldemo.Result{}, fmt.Errorf("unexpected wizard result")
	}
	if resultModel.aborted {
		return machinepooldemo.Result{}, fmt.Errorf("interactive input cancelled")
	}
	return resultModel.result, nil
}

func newWizardModel() wizardModel {
	m := wizardModel{
		step: stepName,
		replicaValidation: &machinepool.ReplicaSizeValidation{
			ClusterVersion: machinepooldemo.DemoClusterVersion,
			MultiAz:        false,
			IsHostedCp:     true,
			Autoscaling:    true,
		},
	}
	m.initStep(stepName)
	return m
}

func (m *wizardModel) initStep(step wizardStep) {
	m.step = step
	m.errMsg = ""

	switch step {
	case stepName:
		m.text = textinput.New()
		m.text.Focus()
		m.text.CharLimit = 64
		m.text.Placeholder = "worker-pool"
	case stepImageType:
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Image Type",
			Options:  machinepooldemo.ImageTypes(),
			Default:  string(cmv1.ImageTypeDefault),
		})
	case stepVersion:
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "OpenShift version",
			Options:  machinepooldemo.Versions(),
			Default:  machinepooldemo.DemoClusterVersion,
			Required: true,
		})
	case stepSelectSubnet:
		m.list = newBoolList("Select subnet for a hosted machine pool", false)
	case stepAvailabilityZone:
		azs := machinepooldemo.AvailabilityZones()
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "AWS availability zone",
			Options:  azs,
			Default:  azs[0],
			Required: true,
		})
	case stepSubnetID:
		opts := machinepooldemo.SubnetOptions()
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Subnet ID",
			Options:  opts,
			Default:  opts[0],
			Required: true,
		})
	case stepAutoscaling:
		m.list = newBoolList("Enable autoscaling", false)
	case stepMinReplicas:
		m.text = textinput.New()
		m.text.Focus()
		m.text.SetValue("2")
	case stepMaxReplicas:
		m.text = textinput.New()
		m.text.Focus()
		m.text.SetValue("4")
	case stepLabels, stepTaints, stepTags, stepCapacityReservationID,
		stepNodeDrainGracePeriod:
		m.text = textinput.New()
		m.text.Focus()
	case stepSecurityGroups:
		m.multi = newMultiSelect(
			optionalQuestionLabel("Additional 'Machine Pool' Security Group IDs"),
			"",
			machinepooldemo.SecurityGroupOptions(),
			0,
		)
	case stepInstanceType:
		types := machinepooldemo.InstanceTypes()
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Instance type",
			Options:  types,
			Default:  types[0],
			Required: true,
		})
	case stepAutorepair:
		m.list = newBoolList("Autorepair", true)
	case stepTuningConfigs:
		m.multi = newMultiSelect(optionalQuestionLabel("Tuning configs"), "", machinepooldemo.TuningConfigNames(), 0)
	case stepCapacityPreference:
		opts := machinepooldemo.CapacityPreferenceOptionsAll()
		if m.result.CapacityReservationID != "" {
			opts = machinepooldemo.CapacityPreferenceOptionsWithID()
		}
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Capacity Reservation Preference",
			Options:  opts,
		})
	case stepKubeletConfig:
		m.multi = newMultiSelect(optionalQuestionLabel("Kubelet config"), "", machinepooldemo.KubeletConfigNames(), 1)
	case stepHTTPTokens:
		opts := machinepooldemo.HttpTokenOptions()
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Configure the use of IMDSv2 for ec2 instances",
			Options:  opts,
			Default:  string(cmv1.Ec2MetadataHttpTokensOptional),
			Required: true,
		})
	case stepRootDiskSize:
		m.text = textinput.New()
		m.text.Focus()
		m.text.SetValue(machinepooldemo.DefaultDiskSize)
	case stepMaxSurge:
		m.text = textinput.New()
		m.text.Focus()
		m.text.SetValue(machinepooldemo.DefaultMaxSurge)
	case stepMaxUnavailable:
		m.text = textinput.New()
		m.text.Focus()
		m.text.SetValue(machinepooldemo.DefaultMaxUnavail)
	}
}

func newBoolList(title string, defaultYes bool) list.Model {
	items := []list.Item{
		boolItem{value: true, label: "Yes"},
		boolItem{value: false, label: "No"},
	}
	defaultIndex := 1
	if defaultYes {
		defaultIndex = 0
	}
	return newSelectList(title, items, defaultIndex)
}

func (m *wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		}
	}

	if m.isMultiStep() {
		return m.updateMulti(msg)
	}
	if m.isListStep() {
		return m.updateList(msg)
	}
	return m.updateText(msg)
}

func (m *wizardModel) isMultiStep() bool {
	return m.step == stepSecurityGroups || m.step == stepTuningConfigs || m.step == stepKubeletConfig
}

func (m *wizardModel) isListStep() bool {
	switch m.step {
	case stepImageType, stepVersion, stepSelectSubnet, stepAvailabilityZone, stepSubnetID,
		stepAutoscaling, stepInstanceType, stepAutorepair, stepCapacityPreference, stepHTTPTokens:
		return true
	default:
		return false
	}
}

func (m *wizardModel) updateMulti(msg tea.Msg) (tea.Model, tea.Cmd) {
	done, values := m.multi.Update(msg)
	if !done {
		return m, nil
	}

	switch m.step {
	case stepSecurityGroups:
		displayValues := append([]string{}, values...)
		for i, sg := range values {
			values[i] = aws.ParseOption(sg)
		}
		m.result.SecurityGroupIDs = values
		recordCmd := m.recordAnswer(optionalQuestionLabel("Additional 'Machine Pool' Security Group IDs"), strings.Join(displayValues, ", "))
		m.initStep(stepTags)
		return m, recordCmd
	case stepTuningConfigs:
		m.result.TuningConfigs = values
		recordCmd := m.recordAnswer(optionalQuestionLabel("Tuning configs"), strings.Join(values, ", "))
		m.initStep(stepCapacityReservationID)
		return m, recordCmd
	case stepKubeletConfig:
		if err := machinepool.ValidateKubeletConfig(values); err != nil {
			m.multi.errMsg = err.Error()
			return m, nil
		}
		m.result.KubeletConfigs = values
		recordCmd := m.recordAnswer(optionalQuestionLabel("Kubelet config"), strings.Join(values, ", "))
		m.initStep(stepHTTPTokens)
		return m, recordCmd
	}
	return m, nil
}

func (m *wizardModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if updated, cmd, handled := maybeStartTypeToFilter(m.list, msg); handled {
		m.list = updated
		return m, cmd
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if key, ok := msg.(tea.KeyMsg); !ok || key.String() != "enter" {
		return m, cmd
	}

	switch m.step {
	case stepImageType:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid image type"
			return m, cmd
		}
		answer := item.value
		if isSkipSelection(item.value) {
			m.result.ImageType = ""
			answer = consts.SkipSelectionOption
		} else {
			m.result.ImageType = item.value
		}
		recordCmd := m.recordAnswer(listPromptLabel(m.list.Title), answer)
		m.initStep(stepVersion)
		return m, mergeCmds(cmd, recordCmd)
	case stepVersion:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid OpenShift version"
			return m, cmd
		}
		m.result.Version = item.value
		recordCmd := m.recordAnswer(listPromptLabel(m.list.Title), item.value)
		m.initStep(stepSelectSubnet)
		return m, mergeCmds(cmd, recordCmd)
	case stepSelectSubnet:
		item, ok := m.list.SelectedItem().(boolItem)
		if !ok {
			m.errMsg = "expected a valid subnet choice"
			return m, cmd
		}
		if item.value {
			m.errMsg = machinepooldemo.MsgGoldenPathSubnet
			return m, cmd
		}
		recordCmd := m.recordAnswer("Select subnet for a hosted machine pool", "No")
		m.initStep(stepAvailabilityZone)
		return m, mergeCmds(cmd, recordCmd)
	case stepAvailabilityZone:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid AWS availability zone"
			return m, cmd
		}
		m.result.AvailabilityZone = item.value
		recordCmd := m.recordAnswer(listPromptLabel(m.list.Title), item.value)
		infoCmd := tea.Println(fmt.Sprintf("There are several subnets for availability zone '%s'", item.value))
		m.initStep(stepSubnetID)
		return m, mergeCmds(cmd, recordCmd, infoCmd)
	case stepSubnetID:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid AWS subnet"
			return m, cmd
		}
		m.result.Subnet = aws.ParseOption(item.value)
		recordCmd := m.recordAnswer(listPromptLabel(m.list.Title), item.value)
		m.initStep(stepAutoscaling)
		return m, mergeCmds(cmd, recordCmd)
	case stepAutoscaling:
		item, ok := m.list.SelectedItem().(boolItem)
		if !ok {
			m.errMsg = "expected a valid autoscaling choice"
			return m, cmd
		}
		if !item.value {
			m.errMsg = machinepooldemo.MsgGoldenPathAutoscaling
			return m, cmd
		}
		m.result.Autoscaling = true
		recordCmd := m.recordAnswer("Enable autoscaling", "Yes")
		m.initStep(stepMinReplicas)
		return m, mergeCmds(cmd, recordCmd)
	case stepInstanceType:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid instance type"
			return m, cmd
		}
		m.result.InstanceType = item.value
		recordCmd := m.recordAnswer(listPromptLabel(m.list.Title), item.value)
		m.initStep(stepAutorepair)
		return m, mergeCmds(cmd, recordCmd)
	case stepAutorepair:
		item, ok := m.list.SelectedItem().(boolItem)
		if !ok {
			m.errMsg = "expected a valid autorepair choice"
			return m, cmd
		}
		m.result.Autorepair = item.value
		label := "No"
		if item.value {
			label = "Yes"
		}
		recordCmd := m.recordAnswer("Autorepair", label)
		m.initStep(stepTuningConfigs)
		return m, mergeCmds(cmd, recordCmd)
	case stepCapacityPreference:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid capacity reservation preference"
			return m, cmd
		}
		answer := item.value
		if isSkipSelection(item.value) {
			m.result.CapacityReservationPref = ""
			answer = consts.SkipSelectionOption
		} else {
			if item.value != "" {
				if err := mpHelpers.ValidateCapacityReservationPreference(item.value, m.result.CapacityReservationID); err != nil {
					m.errMsg = err.Error()
					return m, cmd
				}
			}
			m.result.CapacityReservationPref = item.value
		}
		recordCmd := m.recordAnswer(listPromptLabel(m.list.Title), answer)
		m.initStep(stepKubeletConfig)
		return m, mergeCmds(cmd, recordCmd)
	case stepHTTPTokens:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid http tokens value"
			return m, cmd
		}
		if err := ocm.ValidateHttpTokensValue(item.value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.HTTPTokens = item.value
		recordCmd := m.recordAnswer(listPromptLabel(m.list.Title), item.value)
		m.initStep(stepRootDiskSize)
		return m, mergeCmds(cmd, recordCmd)
	}
	return m, cmd
}

func (m *wizardModel) updateText(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.text, cmd = m.text.Update(msg)
	if key, ok := msg.(tea.KeyMsg); !ok || key.String() != "enter" {
		return m, cmd
	}

	value := m.text.Value()
	switch m.step {
	case stepName:
		if value == "" {
			m.errMsg = "machine pool name is required"
			return m, cmd
		}
		if !machinepool.MachinePoolKeyRE.MatchString(value) {
			m.errMsg = "expected a valid name for the machine pool"
			return m, cmd
		}
		m.result.Name = value
		recordCmd := m.recordAnswer("Machine pool name", value)
		m.initStep(stepImageType)
		return m, mergeCmds(cmd, recordCmd)
	case stepMinReplicas:
		minReplicas, err := strconv.Atoi(value)
		if err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		if err = m.replicaValidation.MinReplicaValidator()(minReplicas); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.MinReplicas = minReplicas
		m.replicaValidation.MinReplicas = minReplicas
		recordCmd := m.recordAnswer("Min replicas", value)
		m.initStep(stepMaxReplicas)
		return m, mergeCmds(cmd, recordCmd)
	case stepMaxReplicas:
		maxReplicas, err := strconv.Atoi(value)
		if err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		if err = m.replicaValidation.MaxReplicaValidator()(maxReplicas); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.MaxReplicas = maxReplicas
		recordCmd := m.recordAnswer("Max replicas", value)
		m.initStep(stepLabels)
		return m, mergeCmds(cmd, recordCmd)
	case stepLabels:
		if err := mpHelpers.LabelValidator(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.Labels = value
		recordCmd := m.recordAnswer(optionalQuestionLabel("Labels"), value)
		m.initStep(stepTaints)
		return m, mergeCmds(cmd, recordCmd)
	case stepTaints:
		if _, err := mpHelpers.ParseTaints(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.Taints = value
		recordCmd := m.recordAnswer(optionalQuestionLabel("Taints"), value)
		m.initStep(stepSecurityGroups)
		return m, mergeCmds(cmd, recordCmd)
	case stepTags:
		if value != "" {
			tags := strings.Split(value, ",")
			if err := aws.UserTagValidator(tags); err != nil {
				m.errMsg = err.Error()
				return m, cmd
			}
			if err := aws.UserTagDuplicateValidator(tags); err != nil {
				m.errMsg = err.Error()
				return m, cmd
			}
		}
		m.result.Tags = value
		recordCmd := m.recordAnswer(optionalQuestionLabel("Tags"), value)
		m.initStep(stepInstanceType)
		return m, mergeCmds(cmd, recordCmd)
	case stepCapacityReservationID:
		m.result.CapacityReservationID = value
		recordCmd := m.recordAnswer(optionalQuestionLabel("Capacity Reservation ID"), value)
		m.initStep(stepCapacityPreference)
		return m, mergeCmds(cmd, recordCmd)
	case stepRootDiskSize:
		if err := interactive.NodePoolRootDiskSizeValidator()(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.RootDiskSize = value
		recordCmd := m.recordAnswer("Root disk size (GiB or TiB)", value)
		m.initStep(stepNodeDrainGracePeriod)
		return m, mergeCmds(cmd, recordCmd)
	case stepNodeDrainGracePeriod:
		if err := mpHelpers.ValidateNodeDrainGracePeriod(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.NodeDrainGracePeriod = value
		recordCmd := m.recordAnswer(optionalQuestionLabel("Node drain grace period"), value)
		m.initStep(stepMaxSurge)
		return m, mergeCmds(cmd, recordCmd)
	case stepMaxSurge:
		if err := mpHelpers.ValidateUpgradeMaxSurgeUnavailable(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.MaxSurge = value
		recordCmd := m.recordAnswer("Max surge", value)
		m.initStep(stepMaxUnavailable)
		return m, mergeCmds(cmd, recordCmd)
	case stepMaxUnavailable:
		if err := mpHelpers.ValidateUpgradeMaxSurgeUnavailable(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.MaxUnavailable = value
		recordCmd := m.recordAnswer("Max unavailable", value)
		m.done = true
		return m, mergeCmds(cmd, recordCmd, tea.Quit)
	}
	return m, cmd
}

func (m *wizardModel) View() string {
	if m.done {
		return ""
	}
	return m.stepView()
}

func (m *wizardModel) stepView() string {
	if m.isMultiStep() {
		return m.multi.View()
	}
	if m.isListStep() {
		view := m.list.View()
		if m.errMsg != "" {
			return view + "\n\n" + m.errMsg + "\n"
		}
		return view + "\n"
	}
	title := m.textStepTitle()
	view := title + "\n\n" + m.text.View() + "\n"
	if m.errMsg != "" {
		view += "\n" + m.errMsg + "\n"
	}
	return view
}

func (m *wizardModel) textStepTitle() string {
	var question string
	optional := false

	switch m.step {
	case stepName:
		question = "Machine pool name"
	case stepMinReplicas:
		question = "Min replicas"
	case stepMaxReplicas:
		question = "Max replicas"
	case stepLabels:
		question = "Labels"
		optional = true
	case stepTaints:
		question = "Taints"
		optional = true
	case stepTags:
		question = "Tags"
		optional = true
	case stepCapacityReservationID:
		question = "Capacity Reservation ID"
		optional = true
	case stepRootDiskSize:
		question = "Root disk size (GiB or TiB)"
	case stepNodeDrainGracePeriod:
		question = "Node drain grace period"
		optional = true
	case stepMaxSurge:
		question = "Max surge"
	case stepMaxUnavailable:
		question = "Max unavailable"
	default:
		return ""
	}

	if optional {
		question = optionalQuestionLabel(question)
	}
	return surveyPromptTitle(question)
}

func isTerminal(out *os.File) bool {
	info, err := out.Stat()
	if err != nil {
		return true
	}
	return info.Mode()&os.ModeCharDevice != 0
}

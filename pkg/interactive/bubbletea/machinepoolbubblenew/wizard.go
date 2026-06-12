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
	"fmt"
	"os"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	checkbox          checkboxList
	completed         []completedAnswer
	errMsg            string
	infoMsg           string
	done              bool
	aborted           bool
	replicaValidation *machinepool.ReplicaSizeValidation
}

// RunWizard collects machine pool settings using a Bubble Tea UX-first wizard.
func RunWizard() (machinepooldemo.Result, error) {
	if !isTerminal(os.Stdout) {
		return machinepooldemo.Result{}, fmt.Errorf("machine pool bubble demo requires an interactive terminal")
	}

	model := newWizardModel()
	final, err := tea.NewProgram(model, tea.WithOutput(os.Stdout)).Run()
	if err != nil {
		return machinepooldemo.Result{}, err
	}
	resultModel, ok := final.(wizardModel)
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
	m.infoMsg = ""

	switch step {
	case stepName:
		m.text = newStyledTextInput()
		m.text.CharLimit = 64
		m.text.Placeholder = "worker-pool"
	case stepImageType:
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Image type",
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
		m.text = newStyledTextInput()
		m.text.SetValue("2")
	case stepMaxReplicas:
		m.text = newStyledTextInput()
		m.text.SetValue("4")
	case stepLabels, stepTaints, stepTags, stepCapacityReservationID, stepNodeDrainGracePeriod:
		m.text = newStyledTextInput()
		m.text.Placeholder = "leave empty to skip"
	case stepSecurityGroups:
		m.checkbox = newCheckboxList(
			"Additional machine pool security group IDs",
			machinepooldemo.SecurityGroupOptions(),
			0,
			true,
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
		m.checkbox = newCheckboxList("Tuning configs", machinepooldemo.TuningConfigNames(), 0, true)
	case stepCapacityPreference:
		opts := machinepooldemo.CapacityPreferenceOptionsAll()
		if m.result.CapacityReservationID != "" {
			opts = machinepooldemo.CapacityPreferenceOptionsWithID()
		}
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Capacity reservation preference",
			Options:  opts,
		})
	case stepKubeletConfig:
		m.checkbox = newCheckboxList("Kubelet config", machinepooldemo.KubeletConfigNames(), 1, true)
	case stepHTTPTokens:
		opts := machinepooldemo.HttpTokenOptions()
		m.list = newROSAOptionList(rosaOptionInput{
			Question: "Configure IMDSv2 for EC2 instances",
			Options:  opts,
			Default:  string(cmv1.Ec2MetadataHttpTokensOptional),
			Required: true,
		})
	case stepRootDiskSize:
		m.text = newStyledTextInput()
		m.text.SetValue(machinepooldemo.DefaultDiskSize)
	case stepMaxSurge:
		m.text = newStyledTextInput()
		m.text.SetValue(machinepooldemo.DefaultMaxSurge)
	case stepMaxUnavailable:
		m.text = newStyledTextInput()
		m.text.SetValue(machinepooldemo.DefaultMaxUnavail)
	}
}

func newStyledTextInput() textinput.Model {
	t := textinput.New()
	t.Focus()
	t.Prompt = "❯ "
	t.PromptStyle = lipgloss.NewStyle().Foreground(accentColor).Bold(true)
	t.TextStyle = lipgloss.NewStyle().Foreground(textColor)
	t.PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)
	t.Cursor.Style = lipgloss.NewStyle().Foreground(accentColor)
	return t
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

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.isListStep() && m.list.FilterState() == list.Filtering {
				break
			}
			if m.isMultiStep() && m.checkbox.list.FilterState() == list.Filtering {
				break
			}
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

func (m wizardModel) updateMulti(msg tea.Msg) (tea.Model, tea.Cmd) {
	done, values, cmd := m.checkbox.Update(msg)
	if !done {
		return m, cmd
	}

	switch m.step {
	case stepSecurityGroups:
		displayValues := append([]string{}, values...)
		for i, sg := range values {
			values[i] = aws.ParseOption(sg)
		}
		m.result.SecurityGroupIDs = values
		m.recordAnswer("Security groups", strings.Join(displayValues, ", "))
		m.initStep(stepTags)
		return m, cmd
	case stepTuningConfigs:
		m.result.TuningConfigs = values
		m.recordAnswer("Tuning configs", strings.Join(values, ", "))
		m.initStep(stepCapacityReservationID)
		return m, cmd
	case stepKubeletConfig:
		if err := machinepool.ValidateKubeletConfig(values); err != nil {
			m.checkbox.errMsg = err.Error()
			return m, nil
		}
		m.result.KubeletConfigs = values
		m.recordAnswer("Kubelet config", strings.Join(values, ", "))
		m.initStep(stepHTTPTokens)
		return m, cmd
	}
	return m, cmd
}

func (m wizardModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if key, ok := msg.(tea.KeyMsg); !ok || key.String() != "enter" {
		return m, cmd
	}
	if m.list.FilterState() == list.Filtering {
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
		m.recordAnswer("Image type", answer)
		m.initStep(stepVersion)
	case stepVersion:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid OpenShift version"
			return m, cmd
		}
		m.result.Version = item.value
		m.recordAnswer("OpenShift version", item.value)
		m.initStep(stepSelectSubnet)
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
		m.recordAnswer("Subnet selection", "No")
		m.initStep(stepAvailabilityZone)
	case stepAvailabilityZone:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid AWS availability zone"
			return m, cmd
		}
		m.result.AvailabilityZone = item.value
		m.recordAnswer("Availability zone", item.value)
		m.infoMsg = fmt.Sprintf("Several subnets are available in %s", item.value)
		m.initStep(stepSubnetID)
	case stepSubnetID:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid AWS subnet"
			return m, cmd
		}
		m.result.Subnet = aws.ParseOption(item.value)
		m.recordAnswer("Subnet ID", item.value)
		m.initStep(stepAutoscaling)
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
		m.recordAnswer("Autoscaling", "Yes")
		m.initStep(stepMinReplicas)
	case stepInstanceType:
		item, ok := m.list.SelectedItem().(optionItem)
		if !ok {
			m.errMsg = "expected a valid instance type"
			return m, cmd
		}
		m.result.InstanceType = item.value
		m.recordAnswer("Instance type", item.value)
		m.initStep(stepAutorepair)
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
		m.recordAnswer("Autorepair", label)
		m.initStep(stepTuningConfigs)
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
		m.recordAnswer("Capacity preference", answer)
		m.initStep(stepKubeletConfig)
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
		m.recordAnswer("IMDSv2", item.value)
		m.initStep(stepRootDiskSize)
	}
	return m, cmd
}

func (m wizardModel) updateText(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.recordAnswer("Machine pool name", value)
		m.initStep(stepImageType)
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
		m.recordAnswer("Min replicas", value)
		m.initStep(stepMaxReplicas)
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
		m.recordAnswer("Max replicas", value)
		m.initStep(stepLabels)
	case stepLabels:
		if err := mpHelpers.LabelValidator(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.Labels = value
		m.recordAnswer("Labels", value)
		m.initStep(stepTaints)
	case stepTaints:
		if _, err := mpHelpers.ParseTaints(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.Taints = value
		m.recordAnswer("Taints", value)
		m.initStep(stepSecurityGroups)
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
		m.recordAnswer("Tags", value)
		m.initStep(stepInstanceType)
	case stepCapacityReservationID:
		m.result.CapacityReservationID = value
		m.recordAnswer("Capacity reservation ID", value)
		m.initStep(stepCapacityPreference)
	case stepRootDiskSize:
		if err := interactive.NodePoolRootDiskSizeValidator()(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.RootDiskSize = value
		m.recordAnswer("Root disk size", value)
		m.initStep(stepNodeDrainGracePeriod)
	case stepNodeDrainGracePeriod:
		if err := mpHelpers.ValidateNodeDrainGracePeriod(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.NodeDrainGracePeriod = value
		m.recordAnswer("Node drain grace period", value)
		m.initStep(stepMaxSurge)
	case stepMaxSurge:
		if err := mpHelpers.ValidateUpgradeMaxSurgeUnavailable(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.MaxSurge = value
		m.recordAnswer("Max surge", value)
		m.initStep(stepMaxUnavailable)
	case stepMaxUnavailable:
		if err := mpHelpers.ValidateUpgradeMaxSurgeUnavailable(value); err != nil {
			m.errMsg = err.Error()
			return m, cmd
		}
		m.result.MaxUnavailable = value
		m.recordAnswer("Max unavailable", value)
		m.done = true
		return m, tea.Quit
	}
	return m, cmd
}

func (m wizardModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(renderHeader(m.step))
	b.WriteString("\n\n")
	if summary := renderSummary(m.completed); summary != "" {
		b.WriteString(summary)
		b.WriteString("\n")
	}
	if m.infoMsg != "" {
		b.WriteString(infoStyle.Render(m.infoMsg))
		b.WriteString("\n\n")
	}
	b.WriteString(m.stepView())
	return b.String()
}

func (m wizardModel) stepView() string {
	if m.isMultiStep() {
		return m.checkbox.View()
	}
	if m.isListStep() {
		view := m.list.View()
		if m.errMsg != "" {
			view += "\n\n" + errorStyle.Render(m.errMsg)
		}
		return view + "\n"
	}
	title, optional, help := m.textStepMeta()
	view := renderTextPrompt(title, optional, help) + "\n\n" + m.text.View() + "\n"
	if m.errMsg != "" {
		view += "\n" + errorStyle.Render(m.errMsg) + "\n"
	}
	return view
}

func (m wizardModel) textStepMeta() (title string, optional bool, help string) {
	switch m.step {
	case stepName:
		return "Machine pool name", false, "Lowercase letters, numbers, and hyphens only"
	case stepMinReplicas:
		return "Minimum replicas", false, ""
	case stepMaxReplicas:
		return "Maximum replicas", false, ""
	case stepLabels:
		return "Labels", true, "Comma-separated key=value pairs"
	case stepTaints:
		return "Taints", true, "Comma-separated key=value:Effect"
	case stepTags:
		return "Tags", true, "Comma-separated key=value pairs"
	case stepCapacityReservationID:
		return "Capacity reservation ID", true, ""
	case stepRootDiskSize:
		return "Root disk size", false, "GiB or TiB, e.g. 100 GiB"
	case stepNodeDrainGracePeriod:
		return "Node drain grace period", true, "e.g. 30 minutes"
	case stepMaxSurge:
		return "Max surge", false, "Nodes added during upgrade"
	case stepMaxUnavailable:
		return "Max unavailable", false, "Nodes unavailable during upgrade"
	default:
		return "", false, ""
	}
}

func (m *wizardModel) recordAnswer(label, value string) {
	m.completed = append(m.completed, completedAnswer{label: label, value: value})
}

func isTerminal(out *os.File) bool {
	info, err := out.Stat()
	if err != nil {
		return true
	}
	return info.Mode()&os.ModeCharDevice != 0
}

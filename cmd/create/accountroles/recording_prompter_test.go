/*
Copyright (c) 2026 Red Hat, Inc.

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

package accountroles

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive"
)

// recordingBoolPrompter records GetBool questions for OCP-43071 flow tests.
// Other Prompter methods are unused in these tests.
type recordingBoolPrompter struct {
	asked    []string
	defaults []any
	answers  map[string]bool
}

func (p *recordingBoolPrompter) GetBool(input interactive.Input) (bool, error) {
	p.asked = append(p.asked, input.Question)
	p.defaults = append(p.defaults, input.Default)
	if answer, ok := p.answers[input.Question]; ok {
		return answer, nil
	}
	if dflt, ok := input.Default.(bool); ok {
		return dflt, nil
	}
	return false, nil
}

func (p *recordingBoolPrompter) GetString(interactive.Input) (string, error) {
	return "", fmt.Errorf("unexpected GetString in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetInt(interactive.Input) (int, error) {
	return 0, fmt.Errorf("unexpected GetInt in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetFloat(interactive.Input) (float64, error) {
	return 0, fmt.Errorf("unexpected GetFloat in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetMultipleOptions(interactive.Input) ([]string, error) {
	return nil, fmt.Errorf("unexpected GetMultipleOptions in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetOption(interactive.Input) (string, error) {
	return "", fmt.Errorf("unexpected GetOption in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetIPNet(interactive.Input) (net.IPNet, error) {
	return net.IPNet{}, fmt.Errorf("unexpected GetIPNet in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetPassword(interactive.Input) (string, error) {
	return "", fmt.Errorf("unexpected GetPassword in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetCert(interactive.Input) (string, error) {
	return "", fmt.Errorf("unexpected GetCert in recordingBoolPrompter")
}
func (p *recordingBoolPrompter) GetOptionMode(*cobra.Command, string, string) (string, error) {
	return "", fmt.Errorf("unexpected GetOptionMode in recordingBoolPrompter")
}

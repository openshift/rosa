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

package interactive

import (
	"net"

	"github.com/spf13/cobra"
)

// Prompter abstracts survey prompts so commands can be tested without a real
// terminal. Same idea as confirmFn in cmd/dlt/cluster.
type Prompter interface {
	GetString(Input) (string, error)
	GetInt(Input) (int, error)
	GetFloat(Input) (float64, error)
	GetMultipleOptions(Input) ([]string, error)
	GetOption(Input) (string, error)
	GetBool(Input) (bool, error)
	GetIPNet(Input) (net.IPNet, error)
	GetPassword(Input) (string, error)
	GetCert(Input) (string, error)
	GetOptionMode(cmd *cobra.Command, mode string, question string) (string, error)
}

// SurveyPrompter is the production Prompter that uses survey.AskOne.
type SurveyPrompter struct{}

// defaultPrompter backs the package-level Get* helpers.
var defaultPrompter Prompter = &SurveyPrompter{}

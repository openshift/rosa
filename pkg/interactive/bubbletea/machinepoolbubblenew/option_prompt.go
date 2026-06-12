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

	"github.com/openshift/rosa/pkg/interactive/consts"
)

type rosaOptionInput struct {
	Question string
	Options  []string
	Default  string
	Required bool
}

// buildROSAOptionPrompt mirrors pkg/interactive.GetOption question/options/default wiring.
func buildROSAOptionPrompt(input rosaOptionInput) (title string, options []string, defaultValue string) {
	options = append([]string{}, input.Options...)
	defaultValue = input.Default

	defaultMessage := ""
	if defaultValue != "" {
		defaultMessage = fmt.Sprintf("default = '%s'", defaultValue)
	}

	question := input.Question
	optionalMessage := ""
	if !input.Required {
		optionalMessage = fmt.Sprintf("optional, choose '%s' to skip selection", consts.SkipSelectionOption)
		options = append([]string{consts.SkipSelectionOption}, options...)
		if defaultValue == "" {
			defaultValue = consts.SkipSelectionOption
		} else {
			optionalMessage += ". The default value will be provided"
		}
	}

	if optionalMessage != "" || defaultMessage != "" {
		question = fmt.Sprintf("%s (%s", question, optionalMessage)
		separator := ""
		if optionalMessage != "" && defaultMessage != "" {
			separator = "; "
		}
		question = fmt.Sprintf("%s%s%s)", question, separator, defaultMessage)
	}

	title = question

	if (defaultValue == "" || !containsString(options, defaultValue)) && len(options) > 0 {
		defaultValue = options[0]
	}

	return title, options, defaultValue
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func isSkipSelection(value string) bool {
	return value == consts.SkipSelectionOption
}

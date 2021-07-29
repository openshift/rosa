/*
Copyright (c) 2020 Red Hat, Inc.

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
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/core"
	"github.com/AlecAivazis/survey/v2/terminal"
)

type Input struct {
	Question string
	Help     string
	Options  []string
	Default  interface{}
	Required bool
}

// Gets user input from the command line
func GetInput(q string) (a string, err error) {
	prompt := &survey.Input{
		Message: fmt.Sprintf("%s:", q),
	}
	survey.AskOne(prompt, &a)
	return
}

// Gets string input from the command line
func GetString(input Input) (a string, err error) {
	dflt, ok := input.Default.(string)
	if !ok {
		dflt = ""
	}
	question := input.Question
	if !input.Required && dflt == "" {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Input{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
		Default: dflt,
	}
	if input.Required {
		err = survey.AskOne(prompt, &a, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &a)
	}
	return
}

// Gets int number input from the command line
func GetInt(input Input) (a int, err error) {
	dflt, ok := input.Default.(int)
	if !ok {
		dflt = 0
	}
	dfltStr := fmt.Sprintf("%d", dflt)
	if dfltStr == "0" {
		dfltStr = ""
	}
	question := input.Question
	if !input.Required && dfltStr == "" {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Input{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
		Default: dfltStr,
	}
	var str string
	if input.Required {
		err = survey.AskOne(prompt, &str, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &str)
	}
	if err != nil {
		return
	}
	if str == "" {
		return
	}
	return parseInt(str)
}

func parseInt(str string) (num int, err error) {
	return strconv.Atoi(str)
}

// Gets float number input from the command line
func GetFloat(input Input) (a float64, err error) {
	dflt, ok := input.Default.(float64)
	if !ok {
		dflt = 0
	}
	dfltStr := fmt.Sprintf("%f", dflt)
	if dfltStr == "0" {
		dfltStr = ""
	}
	question := input.Question
	if !input.Required && dfltStr == "" {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Input{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
	}
	if input.Default != nil {
		prompt.Default = dfltStr
	}
	var str string
	if input.Required {
		err = survey.AskOne(prompt, &str, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &str)
	}
	if err != nil {
		return
	}
	if str == "" {
		return
	}
	return parseFloat(str)
}

func parseFloat(str string) (num float64, err error) {
	return strconv.ParseFloat(str, 64)
}

// Asks for multiple options selection
func GetMultipleOptions(input Input) ([]string, error) {
	var err error
	res := make([]string, 0)
	dflt, ok := input.Default.([]string)
	if !ok {
		dflt = []string{}
	}
	question := input.Question
	if !input.Required && len(dflt) == 0 {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.MultiSelect{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
		Options: input.Options,
		Default: dflt,
	}

	if input.Required {
		err = survey.AskOne(prompt, &res, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &res)
	}
	return res, err
}

// Asks for option selection in the command line
func GetOption(input Input) (a string, err error) {
	dflt, ok := input.Default.(string)
	if !ok {
		dflt = ""
	}
	question := input.Question
	if !input.Required && dflt == "" {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Select{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
		Options: input.Options,
		Default: dflt,
	}
	if input.Required {
		err = survey.AskOne(prompt, &a, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &a)
	}
	return
}

// Asks for true/false value in the command line
func GetBool(input Input) (a bool, err error) {
	dflt, ok := input.Default.(bool)
	if !ok {
		dflt = false
	}
	question := input.Question
	if !input.Required && !dflt {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
		Default: dflt,
	}
	if input.Required {
		err = survey.AskOne(prompt, &a, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &a)
	}
	return
}

// Asks for CIDR value in the command line
func GetIPNet(input Input) (a net.IPNet, err error) {
	dflt, ok := input.Default.(net.IPNet)
	if !ok {
		dflt = net.IPNet{}
	}
	dfltStr := dflt.String()
	if dfltStr == "<nil>" {
		dfltStr = ""
	}
	question := input.Question
	if !input.Required && dfltStr == "" {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Input{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
		Default: dfltStr,
	}
	var str string
	if input.Required {
		err = survey.AskOne(prompt, &str, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &str)
	}
	if err != nil {
		return
	}
	if str == "" {
		return
	}
	_, cidr, err := net.ParseCIDR(str)
	if err != nil {
		return
	}
	if cidr != nil {
		a = *cidr
	}
	return
}

// Gets password input from the command line
func GetPassword(input Input) (a string, err error) {
	question := input.Question
	if !input.Required {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Password{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
	}
	if input.Required {
		err = survey.AskOne(prompt, &a, survey.WithValidator(survey.Required))
	} else {
		err = survey.AskOne(prompt, &a)
	}
	return
}

// Gets path to certificate file from the command line
func GetCert(input Input) (a string, err error) {
	dflt, ok := input.Default.(string)
	if !ok {
		dflt = ""
	}
	question := input.Question
	if !input.Required && dflt == "" {
		question = fmt.Sprintf("%s (optional)", question)
	}
	prompt := &survey.Input{
		Message: fmt.Sprintf("%s:", question),
		Help:    input.Help,
		Default: dflt,
	}
	if input.Required {
		err = survey.AskOne(prompt, &a, survey.WithValidator(survey.Required), survey.WithValidator(certValidator))
	} else {
		err = survey.AskOne(prompt, &a, survey.WithValidator(certValidator))
	}
	return
}

// certValidator validates whether the given filepath is a valid cert file
func certValidator(filepath interface{}) error {
	if filepath == nil {
		return nil
	}
	if s, ok := filepath.(string); ok {
		if s == "" {
			return nil
		}
		validExtension, err := regexp.MatchString("\\.(pem|ca-bundle|ce?rt?|key)$", s)
		if err != nil {
			return err
		}
		if !validExtension {
			return fmt.Errorf("file '%s' does not have a valid file extension", s)
		}
		if _, err := os.Stat(s); !os.IsNotExist(err) {
			// path to file exist
			return nil
		}
		return fmt.Errorf("file '%s' does not exist on the file system", s)
	}
	return fmt.Errorf("can only validate strings, got %v", filepath)
}

var helpTemplate = `{{color "cyan"}}? {{.Message}}
{{range .Steps}}  - {{.}}{{"\n"}}{{end}}{{color "reset"}}`

type Help struct {
	Message string
	Steps   []string
}

func PrintHelp(help Help) error {
	out, _, err := core.RunTemplate(helpTemplate, help)
	if err != nil {
		return err
	}

	fmt.Fprint(terminal.NewAnsiStdout(os.Stdout), out)
	return nil
}

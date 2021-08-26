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

// This file contains functions used to implement the '--interactive' command line option.

package interactive

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"

	"github.com/AlecAivazis/survey/v2"
)

type Validator survey.Validator

var required = survey.Required

var MaxLength = func(length int) Validator {
	return Validator(survey.MaxLength(length))
}

func compose(validators []Validator) survey.Validator {
	surveyValidators := []survey.Validator{}
	for _, validator := range validators {
		surveyValidators = append(surveyValidators, survey.Validator(validator))
	}
	return survey.ComposeValidators(surveyValidators...)
}

// IsURL validates whether the given value is a valid URL
func IsURL(val interface{}) error {
	if s, ok := val.(string); ok {
		if s == "" {
			return nil
		}
		_, err := url.ParseRequestURI(fmt.Sprintf("%v", val))
		return err
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

// IsCert validates whether the given filepath is a valid cert file
func IsCert(filepath interface{}) error {
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

func IsCIDR(val interface{}) error {
	if s, ok := val.(string); ok {
		_, _, err := net.ParseCIDR(s)
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("can only validate strings, got %v", val)
}

func RegExp(restr string) Validator {
	re := regexp.MustCompile(restr)
	return func(val interface{}) error {
		if str, ok := val.(string); ok {
			if str == "" {
				return nil
			}
			if !re.MatchString(str) {
				return fmt.Errorf("%s does not match regular expression %s", str, re.String())
			}
			return nil
		}
		return fmt.Errorf("can only validate strings, got %v", val)
	}
}

func RegExpBoolean(restr string) Validator {
	re := regexp.MustCompile(restr)
	return func(val interface{}) error {
		if boolVal, ok := val.(bool); ok {
			var val string
			if boolVal {
				val = "true"
			} else {
				val = "false"
			}
			if !re.MatchString(val) {
				return fmt.Errorf("%s does not match regular expression %s", val, re.String())
			}
			return nil
		}
		return fmt.Errorf("can only validate boolean values, got %v", val)
	}
}

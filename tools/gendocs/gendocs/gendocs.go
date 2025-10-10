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

package gendocs

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

type Examples struct {
	Items []Example `json:"items"`
}

type Example struct {
	Name        string `json:"name"`
	FullName    string `json:"fullName"`
	Description string `json:"description"`
	Examples    string `json:"examples"`
}

// GenDocs generates Asciidoc documentation for ROSA CLI commands
func GenDocs(cmd *cobra.Command, filename string) error {
	examples := extractExamples(cmd)

	templateContent, err := os.ReadFile("templates/clibyexample/template")
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New("docs").Parse(string(templateContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, examples); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// extractExamples recursively walks the command tree and extracts examples
func extractExamples(cmd *cobra.Command) Examples {
	examples := Examples{
		Items: []Example{},
	}

	// Skip hidden commands
	if cmd.Hidden {
		return examples
	}

	// Add current command if it has examples
	if cmd.Example != "" {
		example := Example{
			Name:        cmd.Name(),
			FullName:    cmd.CommandPath(),
			Description: cmd.Short,
			Examples:    strings.TrimSpace(cmd.Example),
		}
		examples.Items = append(examples.Items, example)
	}

	// Recursively process subcommands
	for _, subCmd := range cmd.Commands() {
		subExamples := extractExamples(subCmd)
		examples.Items = append(examples.Items, subExamples.Items...)
	}

	return examples
}

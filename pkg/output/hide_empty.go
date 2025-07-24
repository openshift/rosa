package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

type TableBuilder struct {
	headers []string
	rows    [][]string
	writer  *tabwriter.Writer
	buffer  *bytes.Buffer
}

func NewTableBuilder() *TableBuilder {
	if !HasHideEmptyFieldsFlag() {
		return &TableBuilder{
			writer: tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0),
		}
	}

	buffer := &bytes.Buffer{}
	return &TableBuilder{
		buffer: buffer,
		writer: tabwriter.NewWriter(buffer, 0, 0, 2, ' ', 0),
	}
}

func (tb *TableBuilder) SetHeaders(headers ...string) {
	tb.headers = headers
}

func (tb *TableBuilder) AddRow(values ...string) {
	tb.rows = append(tb.rows, values)
}

func (tb *TableBuilder) Render() {
	result := tb.buildOutput()
	fmt.Print(result)
}

func (tb *TableBuilder) RenderToString() string {
	return tb.buildOutput()
}

// buildOutput builds the table output string
func (tb *TableBuilder) buildOutput() string {
	if !HasHideEmptyFieldsFlag() {
		// Direct output without filtering
		var outputBuilder strings.Builder
		outputWriter := tabwriter.NewWriter(&outputBuilder, 0, 0, 2, ' ', 0)

		if len(tb.headers) > 0 {
			fmt.Fprintf(outputWriter, "%s\n", strings.Join(tb.headers, "\t"))
		}
		for _, row := range tb.rows {
			fmt.Fprintf(outputWriter, "%s\n", strings.Join(row, "\t"))
		}
		outputWriter.Flush()
		return outputBuilder.String()
	}

	columnsToKeep := make([]bool, len(tb.headers))
	for colIdx := range tb.headers {
		columnsToKeep[colIdx] = false
		// Check if any row has non-empty value for this column
		for _, row := range tb.rows {
			if colIdx < len(row) {
				value := strings.TrimSpace(row[colIdx])
				if value != "" && value != "-" {
					columnsToKeep[colIdx] = true
					break
				}
			}
		}
	}

	// Filter headers
	var filteredHeaders []string
	for i, keep := range columnsToKeep {
		if keep {
			filteredHeaders = append(filteredHeaders, tb.headers[i])
		}
	}

	// Build filtered output
	var outputBuilder strings.Builder
	outputWriter := tabwriter.NewWriter(&outputBuilder, 0, 0, 2, ' ', 0)

	// Write filtered headers
	if len(filteredHeaders) > 0 {
		fmt.Fprintf(outputWriter, "%s\n", strings.Join(filteredHeaders, "\t"))
	}

	// Write filtered rows
	for _, row := range tb.rows {
		var filteredRow []string
		for i, keep := range columnsToKeep {
			if keep && i < len(row) {
				filteredRow = append(filteredRow, row[i])
			}
		}
		if len(filteredRow) > 0 {
			fmt.Fprintf(outputWriter, "%s\n", strings.Join(filteredRow, "\t"))
		}
	}

	outputWriter.Flush()
	return outputBuilder.String()
}

// ProcessTabularOutput processes tabwriter output to hide empty columns when --hide-empty-fields is set
func ProcessTabularOutput(input string) string {
	if !HasHideEmptyFieldsFlag() {
		return input
	}

	// If empty input, return as is
	if strings.TrimSpace(input) == "" {
		return input
	}

	// Split into lines
	lines := strings.Split(strings.TrimRight(input, "\n"), "\n")
	if len(lines) == 0 {
		return input
	}

	// Parse header to get column boundaries
	header := lines[0]
	columnStarts := findColumnStarts(header)
	if len(columnStarts) == 0 {
		return input
	}

	// Determine which columns are empty
	columnsToKeep := make([]bool, len(columnStarts))
	for colIdx := range columnsToKeep {
		columnsToKeep[colIdx] = false
		// Check if any data row has non-empty value for this column
		for lineIdx := 1; lineIdx < len(lines); lineIdx++ {
			colValue := extractColumn(lines[lineIdx], columnStarts, colIdx)
			if !isEmptyValue(strings.TrimSpace(colValue)) {
				columnsToKeep[colIdx] = true
				break
			}
		}
	}

	// If all columns are empty, return just the header
	hasAnyColumn := false
	for _, keep := range columnsToKeep {
		if keep {
			hasAnyColumn = true
			break
		}
	}
	if !hasAnyColumn {
		return header + "\n"
	}

	// Rebuild output with only non-empty columns
	var result strings.Builder

	// Process each line
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result.WriteString("\n")
			continue
		}

		first := true
		for colIdx, keep := range columnsToKeep {
			if !keep {
				continue
			}

			colValue := extractColumn(line, columnStarts, colIdx)
			if !first {
				result.WriteString("\t")
			} else {
				first = false
			}
			result.WriteString(strings.TrimSpace(colValue))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// findColumnStarts finds the starting positions of columns in a header line
func findColumnStarts(header string) []int {
	var starts []int
	inColumn := false

	for i, char := range header {
		if char != ' ' && char != '\t' {
			if !inColumn {
				starts = append(starts, i)
				inColumn = true
			}
		} else {
			inColumn = false
		}
	}

	return starts
}

// extractColumn extracts a column value from a line based on column positions
func extractColumn(line string, columnStarts []int, colIdx int) string {
	if colIdx >= len(columnStarts) {
		return ""
	}

	start := columnStarts[colIdx]
	if start >= len(line) {
		return ""
	}

	var end int
	if colIdx+1 < len(columnStarts) {
		end = columnStarts[colIdx+1]
	} else {
		end = len(line)
	}

	if end > len(line) {
		end = len(line)
	}

	return line[start:end]
}

// isEmptyValue checks if a value is considered empty
func isEmptyValue(value string) bool {
	return value == "" || value == "-"
}

// RemoveEmptyFieldsFromStructuredData removes empty fields from JSON/YAML data
func RemoveEmptyFieldsFromStructuredData(data any) any {
	if !HasHideEmptyFieldsFlag() {
		return data
	}

	switch v := data.(type) {
	case map[string]any:
		result := make(map[string]any)
		for key, value := range v {
			if !isEmptyStructuredValue(value) {
				processed := RemoveEmptyFieldsFromStructuredData(value)
				if !isEmptyStructuredValue(processed) {
					result[key] = processed
				}
			}
		}
		if len(result) == 0 {
			return nil
		}
		return result

	case []any:
		var result []any
		for _, item := range v {
			processed := RemoveEmptyFieldsFromStructuredData(item)
			if !isEmptyStructuredValue(processed) {
				result = append(result, processed)
			}
		}
		return result

	default:
		return v
	}
}

// isEmptyStructuredValue checks if a structured data value is considered empty
func isEmptyStructuredValue(value any) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return v == "" || v == "-"
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case bool:
		return false // Never hide boolean fields
	case float64, int, int64:
		return false // Never hide numeric fields
	default:
		return false
	}
}

// ProcessStructuredOutput processes structured data (JSON/YAML) to remove empty fields if flag is set
func ProcessStructuredOutput(data any) (any, error) {
	if !HasHideEmptyFieldsFlag() {
		return data, nil
	}

	// Convert to JSON then back to any to ensure consistent structure
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	var parsed any
	err = json.Unmarshal(jsonBytes, &parsed)
	if err != nil {
		return nil, err
	}

	return RemoveEmptyFieldsFromStructuredData(parsed), nil
}

package output

import (
	"fmt"
	"text/tabwriter"
)

func CheckIfColumnIsEmpty(columnIdx int, tableData [][]string) bool {
	for _, row := range tableData {
		if columnIdx < len(row) && row[columnIdx] != "" {
			return false
		}
	}
	return true
}

func RemoveEmptyColumns(headers []string, tableData [][]string) [][]string {
	if len(tableData) == 0 {
		return [][]string{headers}
	}

	var newHeaders []string
	var columnsToKeep []int

	for i, header := range headers {
		if !CheckIfColumnIsEmpty(i, tableData) {
			newHeaders = append(newHeaders, header)
			columnsToKeep = append(columnsToKeep, i)
		}
	}

	var result [][]string
	result = append(result, newHeaders)

	for _, row := range tableData {
		var newRow []string
		for _, colIdx := range columnsToKeep {
			if colIdx < len(row) {
				newRow = append(newRow, row[colIdx])
			} else {
				newRow = append(newRow, "")
			}
		}
		result = append(result, newRow)
	}

	return result
}

// BuildTable writes table data to a tabwriter with the specified separator
// The first row in tableData is treated as headers.
// Dynamically builds format strings like "%s\t%s\t%s\n" based on column count.
func BuildTable(writer *tabwriter.Writer, separator string, tableData [][]string) {
	for _, row := range tableData {
		if len(row) == 0 {
			continue
		}

		// Build format string dynamically based on number of columns
		formatString := buildFormatString(len(row), separator)

		// Convert []string to []interface{} for fmt.Fprintf
		args := make([]any, len(row))
		for i, v := range row {
			args[i] = v
		}

		fmt.Fprintf(writer, formatString, args...)
	}
}

// buildFormatString creates a format string with the appropriate number of %s placeholders
func buildFormatString(columnCount int, separator string) string {
	if columnCount == 0 {
		return "\n"
	}

	format := ""
	for i := range columnCount {
		format += "%s"
		if i < columnCount-1 {
			format += separator
		}
	}
	format += "\n"

	return format
}

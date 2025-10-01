package output

import (
	"fmt"
	"strings"
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

	// Keep only non-empty columns
	for i, header := range headers {
		if !CheckIfColumnIsEmpty(i, tableData) {
			newHeaders = append(newHeaders, header)
			columnsToKeep = append(columnsToKeep, i)
		}
	}

	// Build filtered table data
	var newTableData [][]string
	for _, row := range tableData {
		var newRow []string
		for _, col := range columnsToKeep {
			if col < len(row) {
				newRow = append(newRow, row[col])
			} else {
				newRow = append(newRow, "")
			}
		}
		newTableData = append(newTableData, newRow)
	}

	// Combine headers and data as before for backward compatibility
	var result [][]string
	result = append(result, newHeaders)
	result = append(result, newTableData...)

	return result
}

func FilterColumnsWithConditions(headers []string, tableData [][]string, preserveColumn map[int]bool) ([]string, [][]string) {
	var newHeaders []string
	var columnsToKeep []int

	// Keep column if it's marked to be preserved OR if it has data
	for i, header := range headers {
		if preserveColumn[i] || !CheckIfColumnIsEmpty(i, tableData) {
			newHeaders = append(newHeaders, header)
			columnsToKeep = append(columnsToKeep, i)
		}
	}

	// Build filtered table data
	var newTableData [][]string
	for _, row := range tableData {
		var newRow []string
		for _, col := range columnsToKeep {
			if col < len(row) {
				newRow = append(newRow, row[col])
			} else {
				newRow = append(newRow, "")
			}
		}
		newTableData = append(newTableData, newRow)
	}

	return newHeaders, newTableData
}

// RemoveEmptyColumnsWithSeparators removes empty columns and filters the separators list accordingly
func RemoveEmptyColumnsWithSeparators(headers []string, tableData [][]string, separators []string) ([][]string, []string) {
	if len(tableData) == 0 {
		return [][]string{headers}, separators
	}

	var newHeaders []string
	var newSeparators []string
	var columnsToKeep []int

	for i, header := range headers {
		if !CheckIfColumnIsEmpty(i, tableData) {
			newHeaders = append(newHeaders, header)
			columnsToKeep = append(columnsToKeep, i)
			if i < len(separators) {
				newSeparators = append(newSeparators, separators[i])
			}
		}
	}

	var result [][]string
	result = append(result, newHeaders)

	for _, row := range tableData {
		var newRow []string
		for _, col := range columnsToKeep {
			if col < len(row) {
				newRow = append(newRow, row[col])
			} else {
				newRow = append(newRow, "")
			}
		}
		result = append(result, newRow)
	}

	return result, newSeparators
}

func BuildTable(writer *tabwriter.Writer, separator string, tableData [][]string) {
	// Build separators list from separator specification
	if len(tableData) == 0 {
		return
	}

	maxCols := 0
	for _, row := range tableData {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	var separators []string

	if strings.Contains(separator, ",") {
		// Multiple separators specified
		separators = strings.Split(separator, ",")
		// If fewer separators than columns, pad with the last separator (or tab if empty)
		if len(separators) < maxCols {
			padSeparator := "\t"
			if len(separators) > 0 && separators[len(separators)-1] != "" {
				padSeparator = separators[len(separators)-1]
			}
			for len(separators) < maxCols {
				separators = append(separators, padSeparator)
			}
		}
	} else {
		// Single separator - duplicate for all columns (backward compatible)
		separators = make([]string, maxCols)
		for i := range separators {
			separators[i] = separator
		}
	}

	BuildTableWithSeparators(writer, separators, tableData)
}

// BuildTableWithSeparators writes table data using a list of separators
func BuildTableWithSeparators(writer *tabwriter.Writer, separators []string, tableData [][]string) {
	var output string

	for _, row := range tableData {
		for colIdx, col := range row {
			output += col
			// Add separator after column if not the last column
			if colIdx < len(row)-1 {
				// Use the separator at this index if available, otherwise use tab as default
				if colIdx < len(separators) {
					output += separators[colIdx]
				} else {
					output += "\t"
				}
			}
		}

		// Add newline after each row
		output += "\n"
	}

	fmt.Fprint(writer, output)
}

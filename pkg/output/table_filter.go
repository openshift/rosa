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

func RemoveEmptyColumns(headers []string, tableData [][]string) ([]string, [][]string) {
	var newHeaders []string
	var columnsToKeep []int

	for i, header := range headers {
		if !CheckIfColumnIsEmpty(i, tableData) {
			newHeaders = append(newHeaders, header)
			columnsToKeep = append(columnsToKeep, i)
		}
	}

	// Build new table data with only the kept columns
	var newTableData [][]string
	for _, row := range tableData {
		var newRow []string
		for _, colIdx := range columnsToKeep {
			if colIdx < len(row) {
				newRow = append(newRow, row[colIdx])
			} else {
				newRow = append(newRow, "")
			}
		}
		newTableData = append(newTableData, newRow)
	}

	return newHeaders, newTableData
}

type TableConfig struct {
	Separator            string
	HasTrailingSeparator bool
	UseFprintln          bool
}

func PrintTable(writer *tabwriter.Writer, headers []string, tableData [][]string, config TableConfig) {
	headerLine := ""
	for i, header := range headers {
		if i > 0 {
			headerLine += config.Separator
		}
		headerLine += header
	}

	if config.HasTrailingSeparator {
		headerLine += config.Separator
	}

	if config.UseFprintln {
		fmt.Fprintln(writer, headerLine)
	} else {
		fmt.Fprintf(writer, "%s\n", headerLine)
	}

	// Print each data row
	for _, row := range tableData {
		rowLine := ""
		for i, value := range row {
			if i > 0 {
				rowLine += config.Separator
			}
			rowLine += value
		}

		if config.HasTrailingSeparator {
			rowLine += config.Separator
		}

		fmt.Fprintf(writer, "%s\n", rowLine)
	}
}

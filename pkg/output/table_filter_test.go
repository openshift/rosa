package output

import (
	"bytes"
	"strings"
	"text/tabwriter"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Table Filter", func() {
	Describe("BuildTableWithSeparators", func() {
		var (
			buffer *bytes.Buffer
			writer *tabwriter.Writer
		)

		BeforeEach(func() {
			buffer = new(bytes.Buffer)
			writer = tabwriter.NewWriter(buffer, 0, 0, 2, ' ', 0)
		})

		It("Handles normal table data correctly", func() {
			tableData := [][]string{
				{"ID", "NAME", "STATUS"},
				{"1", "Alice", "Active"},
				{"2", "Bob", "Pending"},
			}
			separators := []string{"\t", "\t", "\t"}

			BuildTableWithSeparators(writer, separators, tableData)
			writer.Flush()

			output := buffer.String()
			Expect(output).To(ContainSubstring("ID"))
			Expect(output).To(ContainSubstring("NAME"))
			Expect(output).To(ContainSubstring("STATUS"))
			Expect(output).To(ContainSubstring("Alice"))
			Expect(output).To(ContainSubstring("Active"))
			Expect(output).To(ContainSubstring("Bob"))
			Expect(output).To(ContainSubstring("Pending"))
		})

		It("Handles empty tableData without errors", func() {
			tableData := [][]string{}
			separators := []string{"\t"}

			BuildTableWithSeparators(writer, separators, tableData)
			writer.Flush()

			output := buffer.String()
			Expect(output).To(Equal(""))
		})

		It("Handles single column data", func() {
			tableData := [][]string{
				{"HEADER"},
				{"Value1"},
				{"Value2"},
			}
			separators := []string{"\t"}

			BuildTableWithSeparators(writer, separators, tableData)
			writer.Flush()

			output := buffer.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")
			Expect(len(lines)).To(Equal(3))
			Expect(lines[0]).To(Equal("HEADER"))
			Expect(lines[1]).To(Equal("Value1"))
			Expect(lines[2]).To(Equal("Value2"))
		})

		It("Handles rows with varying column counts", func() {
			tableData := [][]string{
				{"A", "B", "C"},
				{"1"},
				{"2", "3"},
			}
			separators := []string{"\t", "\t", "\t"}

			BuildTableWithSeparators(writer, separators, tableData)
			writer.Flush()

			output := buffer.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")
			Expect(len(lines)).To(Equal(3))
		})

		It("Handles empty strings in cells correctly", func() {
			tableData := [][]string{
				{"ID", "NAME", "STATUS"},
				{"1", "", "Active"},
				{"", "Bob", ""},
			}
			separators := []string{"\t", "\t", "\t"}

			BuildTableWithSeparators(writer, separators, tableData)
			writer.Flush()

			output := buffer.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")
			Expect(len(lines)).To(Equal(3))
			Expect(output).To(ContainSubstring("ID"))
			Expect(output).To(ContainSubstring("NAME"))
			Expect(output).To(ContainSubstring("STATUS"))
			Expect(output).To(ContainSubstring("Active"))
			Expect(output).To(ContainSubstring("Bob"))
		})
	})

	Describe("RemoveEmptyColumns", func() {
		It("Removes columns that are entirely empty", func() {
			headers := []string{"ID", "NAME", "EMPTY", "STATUS"}
			tableData := [][]string{
				{"1", "Alice", "", "Active"},
				{"2", "Bob", "", "Pending"},
			}

			result := RemoveEmptyColumns(headers, tableData)

			Expect(len(result)).To(Equal(3)) // headers + 2 data rows
			Expect(result[0]).To(Equal([]string{"ID", "NAME", "STATUS"}))
			Expect(result[1]).To(Equal([]string{"1", "Alice", "Active"}))
			Expect(result[2]).To(Equal([]string{"2", "Bob", "Pending"}))
		})

		It("Handles empty tableData by returning just headers", func() {
			headers := []string{"ID", "NAME", "STATUS"}
			tableData := [][]string{}

			result := RemoveEmptyColumns(headers, tableData)

			Expect(len(result)).To(Equal(1))
			Expect(result[0]).To(Equal(headers))
		})

		It("Keeps all columns if none are empty", func() {
			headers := []string{"ID", "NAME", "STATUS"}
			tableData := [][]string{
				{"1", "Alice", "Active"},
				{"2", "Bob", "Pending"},
			}

			result := RemoveEmptyColumns(headers, tableData)

			Expect(len(result)).To(Equal(3))
			Expect(result[0]).To(Equal(headers))
			Expect(result[1]).To(Equal([]string{"1", "Alice", "Active"}))
			Expect(result[2]).To(Equal([]string{"2", "Bob", "Pending"}))
		})
	})

	Describe("FilterColumnsWithConditions", func() {
		It("Preserves specified columns even if empty", func() {
			headers := []string{"ID", "NAME", "EMPTY1", "STATUS", "EMPTY2"}
			tableData := [][]string{
				{"1", "Alice", "", "Active", ""},
				{"2", "Bob", "", "Pending", ""},
			}
			preserveColumn := map[int]bool{
				2: true, // Preserve EMPTY1 column even if empty
			}

			resultHeaders, resultData := FilterColumnsWithConditions(headers, tableData, preserveColumn)

			Expect(resultHeaders).To(Equal([]string{"ID", "NAME", "EMPTY1", "STATUS"}))
			Expect(len(resultData)).To(Equal(2))
			Expect(resultData[0]).To(Equal([]string{"1", "Alice", "", "Active"}))
			Expect(resultData[1]).To(Equal([]string{"2", "Bob", "", "Pending"}))
		})

		It("Keeps non-empty columns regardless of preserve flag", func() {
			headers := []string{"ID", "NAME", "STATUS"}
			tableData := [][]string{
				{"1", "Alice", "Active"},
				{"2", "Bob", "Pending"},
			}
			preserveColumn := map[int]bool{} // No columns explicitly preserved

			resultHeaders, resultData := FilterColumnsWithConditions(headers, tableData, preserveColumn)

			Expect(resultHeaders).To(Equal([]string{"ID", "NAME", "STATUS"}))
			Expect(len(resultData)).To(Equal(2))
		})
	})
})

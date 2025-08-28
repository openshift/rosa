package output

import "github.com/spf13/cobra"

var hideEmptyColumnsFlag bool = false

func AddHideEmptyColumnsFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(
		&hideEmptyColumnsFlag,
		"hide-empty-columns",
		false,
		"Hide columns that contain no data",
	)
}

func ShouldHideEmptyColumns() bool {
	return hideEmptyColumnsFlag
}

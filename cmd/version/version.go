package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/qiankunli/workflow/pkg/version"
)

// NewCommand returns the version sub command.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Run the version command",
		Long:  "Run the version command",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Println(version.Get())
		},
	}
}

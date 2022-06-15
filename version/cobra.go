package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	VersionCmd = &cobra.Command{
		Use:   "version",
		Short: "print version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			v := NewBuildInfo()
			fmt.Printf("%#v\n", v.String())
			return nil
		},
	}
)

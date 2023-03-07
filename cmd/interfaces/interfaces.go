package interfaces

import "github.com/spf13/cobra"

var cmdInterfaces = &cobra.Command{
	Use:   "interfaces",
	Short: "Interfaces commands",
}

func Setup(cmd *cobra.Command) {
	cmdInterfaces.AddCommand(cmdList)
	cmd.AddCommand(cmdInterfaces)
}

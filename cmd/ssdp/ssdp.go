package ssdp

import "github.com/spf13/cobra"

var cmdSSDP = &cobra.Command{
	Use:   "ssdp",
	Short: "SSDP commands",
}

func Setup(cmd *cobra.Command) {
	cmdSSDP.AddCommand(cmdListen)
	cmdSSDP.AddCommand(cmdProxy)
	cmdSSDP.AddCommand(cmdDiscover)
	cmd.AddCommand(cmdSSDP)
}

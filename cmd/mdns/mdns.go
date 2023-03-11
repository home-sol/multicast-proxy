package mdns

import "github.com/spf13/cobra"

var cmdMDNS = &cobra.Command{
	Use:   "mdns",
	Short: "MDNS commands",
}

func Setup(cmd *cobra.Command) {
	cmdMDNS.AddCommand(cmdReflector)
	cmd.AddCommand(cmdMDNS)
}

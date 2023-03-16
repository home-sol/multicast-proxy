package cmd

import (
	"github.com/home-sol/multicast-proxy/cmd/ssdp"
	"github.com/spf13/cobra"
)

var root = &cobra.Command{
	Use:     "root",
	Long:    "This ia command for multicast-proxy",
	Version: Version,
}

func Execute() error {
	return root.Execute()
}

func init() {
	ssdp.Setup(root)
	root.AddCommand(cmdServe)
}

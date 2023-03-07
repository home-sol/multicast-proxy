package interfaces

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"
)

var cmdList = &cobra.Command{
	Use:   "list",
	Short: "List Interfaces",
	Long:  "List network interfaces, available on this system",
	RunE: func(cmd *cobra.Command, args []string) error {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		for _, iface := range interfaces {
			fmt.Printf("%s\n", iface.Name)
		}

		return nil
	},
}

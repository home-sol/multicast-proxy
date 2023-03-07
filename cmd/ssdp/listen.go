package ssdp

import (
	"fmt"
	"github.com/home-sol/multicast-proxy/pkg/net/httpu"
	"net"
	"net/http"

	"github.com/home-sol/multicast-proxy/pkg/net/multicast"
	"github.com/spf13/cobra"
)

var cmdListen = &cobra.Command{
	Use:   "listen",
	Short: "Listen for SSDP messages",
	Long:  "Listen for SSDP messages",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		switch len(interfaceNames) {
		case 0:
			var err error
			interfaces, err = net.Interfaces()
			if err != nil {
				return err
			}
		default:
			for _, name := range interfaceNames {
				iface, err := net.InterfaceByName(name)
				if err != nil {
					return err
				}
				interfaces = append(interfaces, *iface)
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		gaddr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
		if err != nil {
			return err
		}
		conn, err := multicast.Listen(gaddr, gaddr, interfaces)
		if err != nil {
			return err
		}

		defer func() {
			if err := conn.Close(); err != nil {
				fmt.Printf("Error closing connection: %v\n", err)
			}
		}()

		return httpu.Serve(cmd.Context(), conn, (httpu.HandlerFunc)(func(r *http.Request) ([]*http.Response, error) {
			fmt.Printf("Request: %v\n", r)
			return nil, nil
		}))
	},
}

var interfaceNames []string

var interfaces []net.Interface

func init() {
	cmdListen.Flags().StringArrayVarP(&interfaceNames, "interface", "i", nil, "Interfaces to listen on by name, e.g. eth0, wlan0, etc.; if not specified, all interfaces will be used")

}

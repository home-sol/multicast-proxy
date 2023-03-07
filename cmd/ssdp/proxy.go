package ssdp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/home-sol/multicast-proxy/pkg/net/httpu"
	"github.com/home-sol/multicast-proxy/pkg/net/multicast"
	"github.com/home-sol/multicast-proxy/pkg/net/ssdp"
	"github.com/spf13/cobra"
)

var cmdProxy = &cobra.Command{
	Use:   "proxy",
	Short: "Proxy for SSDP messages",
	Long:  "Proxy for SSDP messages",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		po.clientInterfaces, err = resolveInterfaceNames(po.clientInterfaceNames)
		if err != nil {
			return err
		}
		po.serverInterfaces, err = resolveInterfaceNames(po.serverInterfaceNames)
		if err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		client, err := httpu.NewClientInterfaces(po.serverInterfaces)
		if err != nil {
			return err
		}

		gaddr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
		if err != nil {
			return err
		}

		conn, err := multicast.Listen(gaddr, gaddr, po.clientInterfaces)
		if err != nil {
			return err
		}

		defer func() {
			if err := conn.Close(); err != nil {
				fmt.Printf("Error closing connection: %v\n", err)
			}
		}()

		return httpu.Serve(cmd.Context(), conn, (httpu.HandlerFunc)(func(r *http.Request) ([]*http.Response, error) {
			if r.Method != "M-SEARCH" {
				return nil, nil
			}

			fmt.Printf("Search: %v\n", r)

			timeout := 5 * time.Second

			timeoutStr := r.Header.Get("MX")
			if timeoutStr != "" {
				timeoutSec, err := strconv.Atoi(timeoutStr)
				if err != nil {
					panic(err)
				}
				timeout = time.Duration(timeoutSec) * time.Second
			}

			searchCtx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			responses, err := ssdp.SSDPRawSearchCtx(searchCtx, client, r.Header.Get("ST"), 1)
			if err != nil {
				fmt.Printf("Error received when performing SSDP Search: %v\n", err)
			}
			fmt.Printf("Received Responses: %d\n", len(responses))

			return responses, nil
		}))
	},
}

func resolveInterfaceNames(interfaceNames []string) ([]net.Interface, error) {
	var interfaces []net.Interface
	switch len(interfaceNames) {
	case 0:
		return net.Interfaces()
	default:
		for _, name := range interfaceNames {
			iface, err := net.InterfaceByName(name)
			if err != nil {
				return nil, err
			}
			interfaces = append(interfaces, *iface)
		}
	}
	return interfaces, nil
}

type proxyOpts struct {
	clientInterfaceNames []string
	serverInterfaceNames []string

	clientInterfaces []net.Interface
	serverInterfaces []net.Interface
}

var po proxyOpts

func init() {
	cmdProxy.Flags().StringArrayVarP(&po.clientInterfaceNames, "client-interface", "c", nil, "Client Interfaces to be proxied")
	cmdProxy.Flags().StringArrayVarP(&po.serverInterfaceNames, "server-interface", "s", nil, "Server Interfaces to be proxied")
}

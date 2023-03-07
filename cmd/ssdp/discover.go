package ssdp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/home-sol/multicast-proxy/pkg/net/httpu"
	"github.com/home-sol/multicast-proxy/pkg/net/ssdp"
	"github.com/spf13/cobra"
)

var cmdDiscover = &cobra.Command{
	Use:   "discover",
	Short: "Discover for SSDP devices",
	Long:  "Discover for SSDP devices",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		intf, err := net.InterfaceByName("enp7s0")
		if err != nil {
			return err
		}
		hc, err := httpu.NewClientInterfaces([]net.Interface{*intf})
		if err != nil {
			return err
		}

		defer func() {
			hcCloseErr := hc.Close()
			if err == nil {
				err = hcCloseErr
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		defer cancel()
		responses, err := ssdp.SSDPRawSearchCtx(ctx, hc, args[0], 3)
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		for _, d := range responses {
			fmt.Printf("%v\n", d)
		}
		return nil
	},
}

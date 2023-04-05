package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/home-sol/multicast-proxy/cmd/ssdp"
	"github.com/spf13/cobra"
)

var (
	cancel context.CancelFunc
)

var root = &cobra.Command{
	Use:     "multicast-proxy",
	Long:    "This is command for multicast-proxy",
	Version: Version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Sigint
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		var ctx context.Context
		ctx, cancel = context.WithCancel(cmd.Context())

		go func() {

			select {
			case <-c:
				cancel()
			}
		}()

		cmd.SetContext(ctx)
	},

	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		cancel()
	},
}

func Execute() error {
	return root.Execute()
}

func init() {
	ssdp.Setup(root)
	root.AddCommand(cmdServe)
}

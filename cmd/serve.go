package cmd

import (
	"github.com/home-sol/multicast-proxy/pkg/net/reflector"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmdServe = &cobra.Command{
	Use:   "serve",
	Short: "Run multicast reflector",
	Long:  `Run multicast reflector, which copies mdns and ssdp packets from one vlan to another`,
	RunE: func(cmd *cobra.Command, args []string) error {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("/etc/home-sol/multicast-proxy/")
		viper.AddConfigPath("$HOME/.multicast-proxy")
		viper.AddConfigPath(".")
		viper.AutomaticEnv()

		err := viper.ReadInConfig()
		if err != nil {
			return err
		}
		var cfg reflector.Config
		err = viper.Unmarshal(&cfg)
		if err != nil {
			return err
		}

		return reflector.Serve(&cfg)
	},
}

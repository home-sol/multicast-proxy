package ssdp

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/home-sol/multicast-proxy/pkg/net/ssdp"
	"github.com/spf13/cobra"
)

var cmdProxy = &cobra.Command{
	Use:   "reflector",
	Short: "SSDP reflector",
	Long:  "SSDP reflector",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := readConfig(args[0])
		if err != nil {
			return fmt.Errorf("could not read configuration: %w", err)
		}
		poolsMap := mapByPool(cfg.Devices)

		deviceToVlanTags := make(map[string][]uint16, len(cfg.Devices))

		for mac, device := range cfg.Devices {
			vlanTags := make([]uint16, 0, len(device.SharedPools))
			for _, pool := range device.SharedPools {
				vlanTags = append(vlanTags, pool)
			}

			deviceToVlanTags[string(mac)] = vlanTags
		}
		//map[uint16][]uint16{
		//	10: {1},
		//	20: {1},
		//}
		return ssdp.Reflector("enp7s0", poolsMap, deviceToVlanTags)
	},
}

type macAddress string

type reflectorConfig struct {
	NetInterface string                       `json:"net_interface"`
	Devices      map[macAddress]bonjourDevice `json:"devices"`
}

type bonjourDevice struct {
	OriginPool  uint16   `json:"origin_pool"`
	SharedPools []uint16 `json:"shared_pools"`
}

func readConfig(path string) (cfg reflectorConfig, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return reflectorConfig{}, err
	}
	err = json.Unmarshal(content, &cfg)
	return cfg, err
}

func mapByPool(devices map[macAddress]bonjourDevice) map[uint16]([]uint16) {
	seen := make(map[uint16]map[uint16]bool)
	poolsMap := make(map[uint16]([]uint16))
	for _, device := range devices {
		for _, pool := range device.SharedPools {
			if _, ok := seen[pool]; !ok {
				seen[pool] = make(map[uint16]bool)
			}
			if _, ok := seen[pool][device.OriginPool]; !ok {
				seen[pool][device.OriginPool] = true
				poolsMap[pool] = append(poolsMap[pool], device.OriginPool)
			}
		}
	}
	return poolsMap
}

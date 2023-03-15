package reflector

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func Serve(cfg *Config) error {
	poolsMap := mapByPool(cfg.Devices)

	return serve(cfg, poolsMap)
}

func serve(cfg *Config, poolsMap map[uint16][]uint16) error {
	// Get a handle on the network interface
	rawTraffic, err := pcap.OpenLive(cfg.NetInterface, 65536, true, time.Second)
	if err != nil {
		return fmt.Errorf("could not open socket on network interface %s: %w", cfg.NetInterface, err)
	}

	// Get the local MAC address, to filter out Bonjour packet generated locally
	intf, err := net.InterfaceByName(cfg.NetInterface)
	if err != nil {
		return err
	}

	intfMACAddress := intf.HardwareAddr

	// Filter tagged bonjour traffic
	filterTemplate := "not (ether src %s) and vlan and ((dst net (239.255.255.250 or ff02::c) and udp dst port 1900) or (dst net (224.0.0.251 or ff02::fb) and udp dst port 5353))"
	err = rawTraffic.SetBPFFilter(fmt.Sprintf(filterTemplate, intfMACAddress))
	if err != nil {
		log.Fatalf("Could not apply filter on network interface: %v", err)
	}

	// Get a channel of Bonjour packets to process
	decoder := gopacket.DecodersByLayerName["Ethernet"]
	source := gopacket.NewPacketSource(rawTraffic, decoder)
	packets := parsePacketsLazily(source)

	// Process packets
	for packet := range packets {
		var vlanTags []uint16
		var hasVlanMapping bool

		if packet.isQuery {
			vlanTags, hasVlanMapping = poolsMap[*packet.vlanTag]
		} else {
			var device Device
			deviceMacAddr := MacAddress(packet.srcMAC.String())
			device, hasVlanMapping = cfg.Devices[deviceMacAddr]
			if hasVlanMapping {
				vlanTags = device.SharedPools
			}
		}

		if !hasVlanMapping {
			continue
		}
		fmt.Printf("%s -> Fwd VLANs: %v\n", packet, vlanTags)
		for _, tag := range vlanTags {
			if err := sendPacket(rawTraffic, &packet, tag, intfMACAddress); err != nil {
				log.Printf("Could not send packet to VLAN %d: %v", tag, err)
			}
		}
	}

	return nil
}

func mapByPool(devices map[MacAddress]Device) map[uint16][]uint16 {
	seen := make(map[uint16]map[uint16]bool)
	poolsMap := make(map[uint16][]uint16)
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

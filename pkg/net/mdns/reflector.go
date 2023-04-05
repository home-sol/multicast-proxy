package mdns

import (
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

func Reflector(intfName string, poolsMap map[uint16][]uint16, deviceToVLanTags map[string][]uint16) error {
	// Get a handle on the network interface
	rawTraffic, err := pcap.OpenLive(intfName, 65536, true, time.Second)
	if err != nil {
		return fmt.Errorf("could not find network interface %s: %w", intfName, err)
	}

	// Get the local MAC address, to filter out Bonjour packet generated locally
	intf, err := net.InterfaceByName(intfName)
	if err != nil {
		return err
	}

	intfMACAddress := intf.HardwareAddr

	// Filter tagged bonjour traffic
	filterTemplate := "not (ether src %s) and vlan and dst net (224.0.0.251 or ff02::fb) and udp dst port 5353"
	err = rawTraffic.SetBPFFilter(fmt.Sprintf(filterTemplate, intfMACAddress))
	if err != nil {
		log.Fatalf("Could not apply filter on network interface: %v", err)
	}

	// Get a channel of Bonjour packets to process
	decoder := gopacket.DecodersByLayerName["Ethernet"]
	source := gopacket.NewPacketSource(rawTraffic, decoder)
	bonjourPackets := parsePacketsLazily(source)

	// Process Bonjours packets
	for bonjourPacket := range bonjourPackets {
		fmt.Println(bonjourPacket.packet.String())
		var vlanTags []uint16
		var hasVlanMapping bool
		// Forward the mDNS query or response to appropriate VLANs
		if bonjourPacket.isDNSQuery {
			vlanTags, hasVlanMapping = poolsMap[*bonjourPacket.vlanTag]
		} else {
			vlanTags, hasVlanMapping = deviceToVLanTags[bonjourPacket.srcMAC.String()]
		}

		if !hasVlanMapping {
			continue
		}
		for _, tag := range vlanTags {
			if err := sendBonjourPacket(rawTraffic, &bonjourPacket, tag, intfMACAddress); err != nil {
				log.Printf("Could not send packet to VLAN %d: %v", tag, err)
			}
		}
	}

	return nil
}

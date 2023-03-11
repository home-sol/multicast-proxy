package ssdp

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type ssdpPacket struct {
	packet      gopacket.Packet
	srcMAC      *net.HardwareAddr
	dstMAC      *net.HardwareAddr
	isIPv6      bool
	vlanTag     *uint16
	isSSDPQuery bool
}

func parsePacketsLazily(source *gopacket.PacketSource) chan ssdpPacket {
	// Process packets, and forward Bonjour traffic to the returned channel

	// Set decoding to Lazy
	source.DecodeOptions = gopacket.DecodeOptions{Lazy: true}

	packetChan := make(chan ssdpPacket, 100)

	go func() {
		for packet := range source.Packets() {
			tag := parseVLANTag(packet)

			// Get source and destination mac addresses
			srcMAC, dstMAC := parseEthernetLayer(packet)

			// Check IP protocol version
			isIPv6 := parseIPLayer(packet)

			payload := parseUDPLayer(packet)

			isSSDPQuery := parseHttpLayer(payload)

			// Pass on the packet for its next adventure
			packetChan <- ssdpPacket{
				packet:      packet,
				vlanTag:     tag,
				srcMAC:      srcMAC,
				dstMAC:      dstMAC,
				isIPv6:      isIPv6,
				isSSDPQuery: isSSDPQuery,
			}
		}
	}()

	return packetChan
}

func parseEthernetLayer(packet gopacket.Packet) (srcMAC, dstMAC *net.HardwareAddr) {
	if parsedEth := packet.Layer(layers.LayerTypeEthernet); parsedEth != nil {
		srcMAC = &parsedEth.(*layers.Ethernet).SrcMAC
		dstMAC = &parsedEth.(*layers.Ethernet).DstMAC
	}
	return
}

func parseVLANTag(packet gopacket.Packet) (tag *uint16) {
	if parsedTag := packet.Layer(layers.LayerTypeDot1Q); parsedTag != nil {
		tag = &parsedTag.(*layers.Dot1Q).VLANIdentifier
	}
	return
}

func parseIPLayer(packet gopacket.Packet) (isIPv6 bool) {
	if parsedIP := packet.Layer(layers.LayerTypeIPv4); parsedIP != nil {
		isIPv6 = false
	}
	if parsedIP := packet.Layer(layers.LayerTypeIPv6); parsedIP != nil {
		isIPv6 = true
	}
	return
}

func parseUDPLayer(packet gopacket.Packet) (payload []byte) {
	if parsedUDP := packet.Layer(layers.LayerTypeUDP); parsedUDP != nil {
		payload = parsedUDP.(*layers.UDP).Payload
	}
	return
}

func parseHttpLayer(payload []byte) bool {
	httpReq, _ := http.ReadRequest(bufio.NewReader(bytes.NewReader(payload)))
	if httpReq.Method == MethodSearch {
		return true
	}
	return false
}

type packetWriter interface {
	WritePacketData([]byte) error
}

func sendSSDPPacket(handle packetWriter, ssdpPacket *ssdpPacket, tag uint16, brMACAddress net.HardwareAddr) error {
	*ssdpPacket.vlanTag = tag
	*ssdpPacket.srcMAC = brMACAddress

	// Network devices may set dstMAC to the local MAC address
	// Rewrite dstMAC to ensure that it is set to the appropriate multicast MAC address
	if ssdpPacket.isIPv6 {
		*ssdpPacket.dstMAC = net.HardwareAddr{0x33, 0x33, 0x00, 0x00, 0x00, 0xFB}
	} else {
		*ssdpPacket.dstMAC = net.HardwareAddr{0x01, 0x00, 0x5E, 0x00, 0x00, 0xFB}
	}

	buf := gopacket.NewSerializeBuffer()
	err := gopacket.SerializePacket(buf, gopacket.SerializeOptions{}, ssdpPacket.packet)
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}
	return handle.WritePacketData(buf.Bytes())
}

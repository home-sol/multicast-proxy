package reflector

import (
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/home-sol/multicast-proxy/pkg/net/ssdp"
)

type packet struct {
	packet   gopacket.Packet
	srcMAC   *net.HardwareAddr
	dstMAC   *net.HardwareAddr
	isIPv6   bool
	vlanTag  *uint16
	isQuery  bool
	srcIP    net.IP
	dstIP    net.IP
	protocol string
	queries  []string
}

func (p packet) String() string {
	return fmt.Sprintf("[%3s] SRC: %1s, DST:%2s, query: %4v", p.protocol, p.srcIP, p.dstIP, p.queries)
}

func parsePacketsLazily(source *gopacket.PacketSource) chan packet {
	// Process packets, and forward Bonjour traffic to the returned channel

	// Set decoding to Lazy
	source.DecodeOptions = gopacket.DecodeOptions{Lazy: true}

	packetChan := make(chan packet, 100)

	go func() {
		for p := range source.Packets() {
			tag := parseVLANTag(p)

			// Get source and destination mac addresses
			srcMAC, dstMAC := parseEthernetLayer(p)

			// Check IP protocol version
			isIPv6, srcIP, dstIP := parseIPLayer(p)

			payload := parseUDPLayer(p)

			var isSSDPPacket, isMDNSPacket, isQuery bool
			var protocol string
			var queries []string

			isSSDPPacket, isQuery, queries = parseSSDPLayer(payload)

			if isSSDPPacket {
				protocol = "SSDP"
			} else {
				isMDNSPacket, isQuery, queries = parseMDNSPayload(payload)
				if isMDNSPacket {
					protocol = "mDNS"
				}
			}

			// Pass on the p for its next adventure
			packetChan <- packet{
				packet:  p,
				vlanTag: tag,
				srcMAC:  srcMAC,
				dstMAC:  dstMAC,
				isIPv6:  isIPv6,
				isQuery: isQuery,

				srcIP:    srcIP,
				dstIP:    dstIP,
				protocol: protocol,
				queries:  queries,
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

func parseIPLayer(packet gopacket.Packet) (bool, net.IP, net.IP) {
	if parsedIP := packet.Layer(layers.LayerTypeIPv4); parsedIP != nil {
		return false, parsedIP.(*layers.IPv4).SrcIP, parsedIP.(*layers.IPv4).DstIP
	}
	if parsedIP := packet.Layer(layers.LayerTypeIPv6); parsedIP != nil {
		return true, parsedIP.(*layers.IPv6).SrcIP, parsedIP.(*layers.IPv6).DstIP
	}
	return false, nil, nil
}

func parseUDPLayer(packet gopacket.Packet) (payload []byte) {
	if parsedUDP := packet.Layer(layers.LayerTypeUDP); parsedUDP != nil {
		payload = parsedUDP.(*layers.UDP).Payload
	}
	return
}

func parseSSDPLayer(payload []byte) (bool, bool, []string) {
	packet := gopacket.NewPacket(payload, ssdp.LayerTypeSSDP, gopacket.Default)
	if parsedSSDP := packet.Layer(ssdp.LayerTypeSSDP); parsedSSDP != nil {
		ssdpPacket := parsedSSDP.(*ssdp.SSDP)
		if ssdpPacket.Method == ssdp.MethodSearch {
			return true, true, []string{ssdpPacket.Headers["ST"]}
		}
		return true, false, nil
	}
	return false, false, nil
}

func parseMDNSPayload(payload []byte) (bool, bool, []string) {
	packet := gopacket.NewPacket(payload, layers.LayerTypeDNS, gopacket.Default)
	if parsedDNS := packet.Layer(layers.LayerTypeDNS); parsedDNS != nil {
		dnsPacket := parsedDNS.(*layers.DNS)
		if !dnsPacket.QR {
			queries := make([]string, len(dnsPacket.Questions))
			for i, question := range dnsPacket.Questions {
				queries[i] = string(question.Name)
			}
			return true, !dnsPacket.QR, queries
		}
		return true, !dnsPacket.QR, nil
	}
	return false, false, nil
}

type packetWriter interface {
	WritePacketData([]byte) error
}

func sendPacket(handle packetWriter, packet *packet, tag uint16, brMACAddress net.HardwareAddr) error {
	*packet.vlanTag = tag
	*packet.srcMAC = brMACAddress

	// Network devices may set dstMAC to the local MAC address
	// Rewrite dstMAC to ensure that it is set to the appropriate multicast MAC address
	if packet.isIPv6 {
		*packet.dstMAC = net.HardwareAddr{0x33, 0x33, 0x00, 0x00, 0x00, 0xFB}
	} else {
		*packet.dstMAC = net.HardwareAddr{0x01, 0x00, 0x5E, 0x00, 0x00, 0xFB}
	}

	buf := gopacket.NewSerializeBuffer()
	err := gopacket.SerializePacket(buf, gopacket.SerializeOptions{}, packet.packet)
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}
	return handle.WritePacketData(buf.Bytes())
}

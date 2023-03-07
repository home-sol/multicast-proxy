package multicast

import (
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/net/ipv4"
)

func Listen(lAddr *net.UDPAddr, rAddr *net.UDPAddr, ifList []net.Interface) (*ipv4.PacketConn, error) {
	conn, err := net.ListenUDP("udp4", lAddr)
	if err != nil {
		return nil, err
	}

	pconn, err := joinGroupIPv4(conn, ifList, rAddr)
	if err != nil {
		if err := conn.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to close UDP connection: %s\n", err)
		}
		return nil, err
	}

	return pconn, nil
}

func joinGroupIPv4(conn *net.UDPConn, iflist []net.Interface, gaddr net.Addr) (*ipv4.PacketConn, error) {
	wrap := ipv4.NewPacketConn(conn)
	err := wrap.SetMulticastLoopback(true)
	if err != nil {
		return nil, err
	}
	// add interfaces to multicast group.
	joined := 0
	for _, ifi := range iflist {
		if err := wrap.JoinGroup(&ifi, gaddr); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to join group %s on %s: %s\n", gaddr.String(), ifi.Name, err)
			continue
		}
		joined++
		fmt.Printf("joined group %s on %s (#%d)\n", gaddr.String(), ifi.Name, ifi.Index)
	}
	if joined == 0 {
		return nil, errors.New("no interfaces had joined to group")
	}
	return wrap, nil
}

package multicast

import (
	"fmt"
	"net"
)

// Ipv4Address returns the IPv4 addresses of the interfaces that support
// multicast.
func Ipv4Address(intfs []net.Interface) ([]string, error) {
	// Find the set of addresses to listen on.
	var addrs []string
	for _, intf := range intfs {
		if intf.Flags&net.FlagMulticast == 0 || intf.Flags&net.FlagLoopback != 0 || intf.Flags&net.FlagUp == 0 {
			// Does not support multicast or is a loopback address.
			continue
		}
		ifaceAddrs, err := intf.Addrs()
		if err != nil {
			return nil, fmt.Errorf("finding addresses on interface %s: %w", intf.Name, err)
		}
		for _, netAddr := range ifaceAddrs {
			addr, ok := netAddr.(*net.IPNet)
			if !ok {
				// Not an IPNet address.
				continue
			}
			// if IPv6 representation of IPv4 address
			if addr.IP.To4() == nil {
				// Not IPv4.
				continue
			}
			addrs = append(addrs, addr.IP.String())
		}
	}
	return addrs, nil
}

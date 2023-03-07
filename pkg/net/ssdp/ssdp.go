package ssdp

const (
	MethodSearch = "M-SEARCH"
	MethodNotify = "NOTIFY"
	SsdpDiscover = `"ssdp:discover"`

	NtsAlive  = `ssdp:alive`
	NtsByebye = `ssdp:byebye`
	NtsUpdate = `ssdp:update`

	SearchPort = 1900
	UDP4Addr   = "239.255.255.250:1900"

	// SsdpAll is a value for searchTarget that searches for all devices and services.
	SsdpAll = "ssdp:all"
	// UPNPRootDevice is a value for searchTarget that searches for all root devices.
	UPNPRootDevice = "upnp:rootdevice"
)

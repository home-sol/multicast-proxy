package ssdp

import (
	"net"
	"net/http"
	"regexp"
	"strconv"
)

type SSDPMessage struct {
	// From is a sender of this message
	From net.Addr

	// Type is a property of "NT"
	Type string

	rawHeader http.Header
}

// Header returns all properties in alive message.
func (m SSDPMessage) Header() http.Header {
	return m.rawHeader
}

// Get returns a property value by name.
func (m SSDPMessage) Get(name string) string {
	return m.rawHeader.Get(name)
}

// AliveMessage represents SSDP's ssdp:alive message.
type AliveMessage struct {
	SSDPMessage

	// USN is a property of "USN"
	USN string

	// Location is a property of "LOCATION"
	Location string

	// Server is a property of "SERVER"
	Server string

	maxAge *int
}

// MaxAge extracts "max-age" value from "CACHE-CONTROL" property.
func (m *AliveMessage) MaxAge() int {
	if m.maxAge == nil {
		m.maxAge = new(int)
		*m.maxAge = extractMaxAge(m.Get("CACHE-CONTROL"), -1)
	}
	return *m.maxAge
}

// ByeMessage represents SSDP's ssdp:byebye message.
type ByeMessage struct {
	SSDPMessage

	// USN is a property of "USN"
	USN string
}

// SearchMessage represents SSDP's ssdp:discover message.
type SearchMessage struct {
	SSDPMessage
}

var rxMaxAge = regexp.MustCompile(`\bmax-age\s*=\s*(\d+)\b`)

func extractMaxAge(s string, value int) int {
	v := value
	if m := rxMaxAge.FindStringSubmatch(s); m != nil {
		i64, err := strconv.ParseInt(m[1], 10, 32)
		if err == nil {
			v = int(i64)
		}
	}
	return v
}

package ssdp

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	maxExpiryTimeSeconds = 24 * 60 * 60
)

var (
	maxAgeRx = regexp.MustCompile("max-age= *([0-9]+)")
)

type Entry struct {
	// The address that the entry data was actually received from.
	RemoteAddr string
	// Unique Service Name. Identifies a unique instance of a device or service.
	USN string
	// Notfication Type. The type of device or service being announced.
	NT string
	// Server's self-identifying string.
	Server string
	Host   string
	// Location of the UPnP root device description.
	Location url.URL
	// Wakeup MAC address.
	Wakeup string

	// Despite BOOTID,CONFIGID being required fields, apparently they are not
	// always set by devices. Set to -1 if not present.

	BootID   int32
	ConfigID int32

	SearchPort uint16

	// When the last update was received for this entry identified by this USN.
	LastUpdate time.Time
	// When the last update's cached values are advised to expire.
	CacheExpiry time.Time
}

func newEntryFromRequest(r *http.Request) (*Entry, error) {
	now := time.Now()
	expiryDuration, err := parseCacheControlMaxAge(r.Header.Get("CACHE-CONTROL"))
	if err != nil {
		return nil, fmt.Errorf("ssdp: error parsing CACHE-CONTROL max age: %v", err)
	}

	loc, err := url.Parse(r.Header.Get("LOCATION"))
	if err != nil {
		return nil, fmt.Errorf("ssdp: error parsing entry Location URL: %v", err)
	}

	bootID, err := parseUpnpIntHeader(r.Header, "BOOTID.UPNP.ORG", -1)
	if err != nil {
		return nil, err
	}
	configID, err := parseUpnpIntHeader(r.Header, "CONFIGID.UPNP.ORG", -1)
	if err != nil {
		return nil, err
	}
	searchPort, err := parseUpnpIntHeader(r.Header, "SEARCHPORT.UPNP.ORG", SearchPort)
	if err != nil {
		return nil, err
	}

	if searchPort < 1 || searchPort > 65535 {
		return nil, fmt.Errorf("ssdp: search port %d is out of range", searchPort)
	}

	return &Entry{
		RemoteAddr:  r.RemoteAddr,
		USN:         r.Header.Get("USN"),
		NT:          r.Header.Get("NT"),
		Server:      r.Header.Get("SERVER"),
		Host:        r.Header.Get("HOST"),
		Wakeup:      r.Header.Get("WAKEUP"),
		Location:    *loc,
		BootID:      bootID,
		ConfigID:    configID,
		SearchPort:  uint16(searchPort),
		LastUpdate:  now,
		CacheExpiry: now.Add(expiryDuration),
	}, nil
}

func parseCacheControlMaxAge(cc string) (time.Duration, error) {
	matches := maxAgeRx.FindStringSubmatch(cc)
	if len(matches) != 2 {
		return 0, fmt.Errorf("did not find exactly one max-age in cache control header: %q", cc)
	}
	expirySeconds, err := strconv.ParseInt(matches[1], 10, 16)
	if err != nil {
		return 0, err
	}
	if expirySeconds < 1 || expirySeconds > maxExpiryTimeSeconds {
		return 0, fmt.Errorf("rejecting bad expiry time of %d seconds", expirySeconds)
	}
	return time.Duration(expirySeconds) * time.Second, nil
}

// parseUpnpIntHeader is intended to parse the
// {BOOT,CONFIGID,SEARCHPORT}.UPNP.ORG header fields. It returns the def if
// the head is empty or missing.
func parseUpnpIntHeader(headers http.Header, headerName string, def int32) (int32, error) {
	s := headers.Get(headerName)
	if s == "" {
		return def, nil
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("ssdp: could not parse header %s: %v", headerName, err)
	}
	return int32(v), nil
}

const (
	EventAlive = EventType(iota)
	EventUpdate
	EventByeBye
)

type EventType int8

func (et EventType) String() string {
	switch et {
	case EventAlive:
		return "EventAlive"
	case EventUpdate:
		return "EventUpdate"
	case EventByeBye:
		return "EventByeBye"
	default:
		return fmt.Sprintf("EventUnknown(%d)", int8(et))
	}
}

type Update struct {
	// The USN of the service.ntsUpdate  gv
	USN string
	// What happened.
	EventType EventType
	// The entry, which is nil if the service was not known and
	// EventType==EventByeBye. The contents of this must not be modified as it is
	// shared with the registry and other listeners. Once created, the Registry
	// does not modify the Entry value - any updates are replaced with a new
	// Entry value.
	Entry *Entry
}

type Registry struct {
	lock  sync.Mutex
	byUSN map[string]*Entry

	listenersLock sync.RWMutex
	listeners     map[chan<- Update]struct{}
}

func NewRegistry() *Registry {
	return &Registry{
		byUSN:     make(map[string]*Entry),
		listeners: make(map[chan<- Update]struct{}),
	}
}

func (reg *Registry) AddListener(c chan<- Update) {
	reg.listenersLock.Lock()
	defer reg.listenersLock.Unlock()
	reg.listeners[c] = struct{}{}
}

func (reg *Registry) RemoveListener(c chan<- Update) {
	reg.listenersLock.Lock()
	defer reg.listenersLock.Unlock()
	delete(reg.listeners, c)
}

func (reg *Registry) sendUpdate(u Update) {
	reg.listenersLock.RLock()
	defer reg.listenersLock.RUnlock()
	for c := range reg.listeners {
		c <- u
	}
}

// ServeMessage implements httpu.Handler, and uses SSDP NOTIFY requests to
// maintain the registry of devices and services.
func (reg *Registry) ServeMessage(r *http.Request) {
	if r.Method != MethodNotify {
		return
	}

	nts := r.Header.Get("nts")

	var err error
	switch nts {
	case NtsAlive:
		err = reg.handleNTSAlive(r)
	case NtsUpdate:
		err = reg.handleNTSUpdate(r)
	case NtsByebye:
		err = reg.handleNTSByebye(r)
	default:
		err = fmt.Errorf("unknown NTS value: %q", nts)
	}
	if err != nil {
		log.Printf("goupnp/ssdp: failed to handle %s message from %s: %v", nts, r.RemoteAddr, err)
	}
}

func (reg *Registry) handleNTSAlive(r *http.Request) error {
	entry, err := newEntryFromRequest(r)
	if err != nil {
		return err
	}

	reg.lock.Lock()
	reg.byUSN[entry.USN] = entry
	reg.lock.Unlock()

	reg.sendUpdate(Update{
		USN:       entry.USN,
		EventType: EventAlive,
		Entry:     entry,
	})

	return nil
}

func (reg *Registry) handleNTSUpdate(r *http.Request) error {
	entry, err := newEntryFromRequest(r)
	if err != nil {
		return err
	}
	nextBootID, err := parseUpnpIntHeader(r.Header, "NEXTBOOTID.UPNP.ORG", -1)
	if err != nil {
		return err
	}
	entry.BootID = nextBootID

	reg.lock.Lock()
	reg.byUSN[entry.USN] = entry
	reg.lock.Unlock()

	reg.sendUpdate(Update{
		USN:       entry.USN,
		EventType: EventUpdate,
		Entry:     entry,
	})

	return nil
}

func (reg *Registry) handleNTSByebye(r *http.Request) error {
	usn := r.Header.Get("USN")

	reg.lock.Lock()
	entry := reg.byUSN[usn]
	delete(reg.byUSN, usn)
	reg.lock.Unlock()

	reg.sendUpdate(Update{
		USN:       usn,
		EventType: EventByeBye,
		Entry:     entry,
	})

	return nil
}

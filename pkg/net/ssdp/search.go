package ssdp

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/home-sol/multicast-proxy/pkg/net/httpu"
)

// SSDPRawSearchCtx performs a fairly raw SSDP search request, and returns the
// unique response(s) that it receives. Each response has the requested
// searchTarget, a USN, and a valid location. maxWaitSeconds states how long to
// wait for responses in seconds, and must be a minimum of 1 (the
// implementation waits an additional 100ms for responses to arrive), 2 is a
// reasonable value for this. numSends is the number of requests to send - 3 is
// a reasonable value for this.
func SSDPRawSearchCtx(ctx context.Context, client httpu.Client, searchTarget string, numSends int) ([]*http.Response, error) {
	maxWaitSeconds := 4
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		maxWaitSeconds = int(deadline.Sub(time.Now()).Seconds())
	}
	req := (&http.Request{
		Method: MethodSearch,
		// TODO: Support both IPv4 and IPv6.
		Host: UDP4Addr,
		URL:  &url.URL{Opaque: "*"},
		Header: http.Header{
			// Putting headers in here avoids them being title-cased.
			// (The UPnP discovery protocol uses case-sensitive headers)
			"HOST": []string{UDP4Addr},
			"MX":   []string{strconv.FormatInt(int64(maxWaitSeconds), 10)},
			"MAN":  []string{SsdpDiscover},
			"ST":   []string{searchTarget},
		},
	}).WithContext(ctx)
	allResponses, err := client.Do(ctx, req, numSends)
	if err != nil {
		return nil, err
	}

	isExactSearch := searchTarget != SsdpAll && searchTarget != UPNPRootDevice

	seenIDs := make(map[string]bool, len(allResponses))
	var responses []*http.Response
	for _, response := range allResponses {
		if response.StatusCode != 200 {
			log.Printf("ssdp: got response status code %q in search response", response.Status)
			continue
		}
		if st := response.Header.Get("ST"); isExactSearch && st != searchTarget {
			continue
		}
		usn := response.Header.Get("USN")
		loc, err := response.Location()
		if err != nil {
			// No usable location in search response - discard.
			continue
		}
		id := loc.String() + "\x00" + usn
		if _, alreadySeen := seenIDs[id]; !alreadySeen {
			seenIDs[id] = true
			responses = append(responses, response)
		}
	}

	return responses, nil
}

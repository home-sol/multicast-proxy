package httpu

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/home-sol/multicast-proxy/pkg/net/multicast"
	"golang.org/x/sync/errgroup"
)

type multiClient struct {
	delegates []Client
}

// NewClientInterfaces creates a HTTPU client that multiplexes to all multicast-capable
// IPv4 addresses on the host. Returns a function to clean up once the client is
// no longer required.
func NewClientInterfaces(interfaceList []net.Interface) (Client, error) {
	ipv4Addresses, err := multicast.Ipv4Address(interfaceList)
	if err != nil {

		return nil, fmt.Errorf("requesting host IPv4 addresses: %w", err)
	}

	delegates := make([]Client, 0, len(ipv4Addresses))
	for _, addr := range ipv4Addresses {
		c, err := NewClientAddr(addr)
		if err != nil {

			return nil, fmt.Errorf("creating HTTPU client for address %s: %w", addr, err)
		}
		delegates = append(delegates, c)
	}

	if len(delegates) == 1 {
		return delegates[0], nil
	}

	return NewMultiClient(delegates), nil
}

func NewMultiClient(delegates []Client) Client {
	return &multiClient{delegates: delegates}
}

func (mc multiClient) Close() error {
	for _, d := range mc.delegates {
		if err := d.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (mc multiClient) Do(ctx context.Context, req *http.Request, numSends int) ([]*http.Response, error) {
	results := make(chan []*http.Response)
	tasks, taskCtx := errgroup.WithContext(ctx)
	tasks.Go(func() error {
		defer close(results)
		return mc.sendRequests(taskCtx, results, req, numSends)
	})

	var responses []*http.Response
	tasks.Go(func() error {
		for rs := range results {
			responses = append(responses, rs...)
		}
		return nil
	})

	return responses, tasks.Wait()
}

func (mc multiClient) sendRequests(ctx context.Context, results chan []*http.Response, req *http.Request, numSends int) error {
	tasks, taskCtx := errgroup.WithContext(ctx)
	for _, d := range mc.delegates {
		d := d
		tasks.Go(func() error {
			responses, err := d.Do(taskCtx, req, numSends)
			if err != nil {
				return err
			}
			results <- responses
			return nil
		})
	}
	return tasks.Wait()
}

package httpu

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const DefaultTimeout = 10 * time.Second

const LocalAddressHeader = "X-local-address"

// Client is an interface for sending HTTP over UDP requests and receive responses.
type Client interface {
	io.Closer

	// Do sends the request and returns the responses. numSends controls the number of times the request is sent.
	Do(ctx context.Context, req *http.Request, numSends int) ([]*http.Response, error)
}

// HttpUClient is a client dealing with HTTP over UDP. Its typical function is for HTTPMU, and particularly SSDP.
type client struct {
	connLock sync.Mutex // Protects use of conn.
	conn     net.PacketConn
}

func NewHTTPUClient() (Client, error) {
	conn, err := net.ListenPacket("udp", ":0")
	if err != nil {
		return nil, err
	}
	return &client{conn: conn}, nil
}

// NewClientAddr creates a new HTTPUClient which will broadcast packets
// from the specified address, opening up a new UDP socket for the purpose
func NewClientAddr(addr string) (Client, error) {
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, errors.New("invalid listening address")
	}
	conn, err := net.ListenPacket("udp", ip.String()+":0")
	if err != nil {
		return nil, err
	}
	return &client{conn: conn}, nil
}

// Close shuts down the client. The client will no longer be useful following this.
func (c *client) Close() error {
	c.connLock.Lock()
	defer c.connLock.Unlock()
	return c.conn.Close()
}

func (c *client) Do(ctx context.Context, req *http.Request, numSends int) ([]*http.Response, error) {
	c.connLock.Lock()
	defer c.connLock.Unlock()

	var requestBuf bytes.Buffer

	err := WriteRequest(&requestBuf, req)
	if err != nil {
		return nil, err
	}

	destAddr, err := net.ResolveUDPAddr("udp", req.Host)
	if err != nil {
		return nil, err
	}

	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		err = c.conn.SetDeadline(deadline)
		if err != nil {
			return nil, err
		}
	} else {
		c.conn.SetDeadline(time.Now().Add(DefaultTimeout))
	}

	// Send request.
	for i := 0; i < numSends; i++ {
		if n, err := c.conn.WriteTo(requestBuf.Bytes(), destAddr); err != nil {
			return nil, err
		} else if n < len(requestBuf.Bytes()) {
			return nil, fmt.Errorf("httpu: wrote %d bytes rather than full %d in request",
				n, len(requestBuf.Bytes()))
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Await for responses until timeout.
	var responses []*http.Response
	responseBytes := make([]byte, 2048)
	for {
		// 2048 bytes should be sufficient for most networks.
		n, _, err := c.conn.ReadFrom(responseBytes)
		if err != nil {
			if err, ok := err.(net.Error); ok {
				if err.Timeout() {
					break
				}
				continue
			}
			return nil, err
		}
		// Parse response.
		response, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(responseBytes[:n])), req)
		if err != nil {
			log.Printf("httpu: error while parsing response: %v", err)
			continue
		}

		// Set the related local address used to discover the device.
		if a, ok := c.conn.LocalAddr().(*net.UDPAddr); ok {
			response.Header.Add(LocalAddressHeader, a.IP.String())
		}

		responses = append(responses, response)
	}

	return responses, nil
}

func WriteRequest(wr io.Writer, req *http.Request) error {
	method := req.Method
	if method == "" {
		method = "GET"
	}

	if _, err := fmt.Fprintf(wr, "%s %s HTTP/1.1\r\n", method, req.URL.RequestURI()); err != nil {
		return err
	}

	for k, valList := range req.Header {
		for _, v := range valList {
			if _, err := fmt.Fprintf(wr, "%s: %s\r\n", strings.ToUpper(k), v); err != nil {
				return err
			}
		}
	}
	if _, err := wr.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

func WriteResponse(wr io.Writer, res *http.Response) error {
	if _, err := fmt.Fprintf(wr, "HTTP/1.1 %s\r\n", res.Status); err != nil {
		return err
	}

	for k, valList := range res.Header {
		if k == "X-Local-Address" {
			continue
		}
		for _, v := range valList {
			if _, err := fmt.Fprintf(wr, "%s: %s\r\n", strings.ToUpper(k), v); err != nil {
				return err
			}
		}
	}
	if _, err := wr.Write([]byte{'\r', '\n'}); err != nil {
		return err
	}
	return nil
}

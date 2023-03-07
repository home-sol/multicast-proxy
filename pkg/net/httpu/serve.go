package httpu

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sync"

	"golang.org/x/net/ipv4"
	"golang.org/x/sync/errgroup"
)

const (
	DefaultMaxMessageBytes = 2048
)

var (
	trailingWhitespaceRx = regexp.MustCompile(" +\r\n")
	crlf                 = []byte("\r\n")
)

// Handler is the interface by which received HTTPU messages are passed to
// handling code.
type Handler interface {
	// ServeMessage is called for each HTTPU message received. peerAddr contains
	// the address that the message was received from.
	ServeMessage(r *http.Request) ([]*http.Response, error)
}

// HandlerFunc is a function-to-Handler adapter.
type HandlerFunc func(r *http.Request) ([]*http.Response, error)

func (f HandlerFunc) ServeMessage(r *http.Request) ([]*http.Response, error) {
	return f(r)
}

type server struct {
	Handler         Handler
	MaxMessageBytes int
}

// Serve messages received on the given packet listener to the given handler.
func Serve(ctx context.Context, conn *ipv4.PacketConn, handler Handler) error {
	srv := server{
		Handler:         handler,
		MaxMessageBytes: DefaultMaxMessageBytes,
	}
	return srv.Serve(ctx, conn)
}

func (srv *server) Serve(ctx context.Context, conn *ipv4.PacketConn) error {
	maxMessageBytes := DefaultMaxMessageBytes
	if srv.MaxMessageBytes != 0 {
		maxMessageBytes = srv.MaxMessageBytes
	}

	bufPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, maxMessageBytes)
		},
	}
	tasks, _ := errgroup.WithContext(ctx)
	defer tasks.Wait()
	for {
		buf := bufPool.Get().([]byte)
		n, _, peerAddr, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}

		tasks.Go(func() error {
			defer bufPool.Put(buf)
			// At least one router's UPnP implementation has added a trailing space
			// after "HTTP/1.1" - trim it.
			reqBuf := trailingWhitespaceRx.ReplaceAllLiteral(buf[:n], crlf)

			req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(reqBuf)))
			if err != nil {
				log.Printf("httpu: Failed to parse request: %v", err)
				return err
			}
			req.RemoteAddr = peerAddr.String()
			responses, err := srv.Handler.ServeMessage(req)
			// No need to call req.Body.Close - underlying reader is bytes.Buffer.
			if err != nil {
				log.Printf("httpu: Failed to handle request: %v", err)
				return nil
			}
			wr := bytes.Buffer{}
			for _, resp := range responses {
				wr.Reset()
				if err := WriteResponse(&wr, resp); err != nil {
					fmt.Printf("Error while encoding response: %v\n", err)
				}
				if _, err := conn.WriteTo(wr.Bytes(), nil, peerAddr); err != nil {
					fmt.Printf("Error writing response: %v\n", err)
					return nil
				}
			}
			return nil
		})
	}
}

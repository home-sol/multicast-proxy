package ssdp

import (
	"bufio"
	"bytes"
	"errors"
	"regexp"
	"strconv"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

var (
	LayerTypeSSDP = gopacket.RegisterLayerType(1001, gopacket.LayerTypeMetadata{Name: "SSDP", Decoder: gopacket.DecodeFunc(decodeSSDP)})
)

type SSDP struct {
	layers.BaseLayer

	Method     string
	URL        string
	Headers    map[string]string
	StatusCode int
	Status     string
}

func (s *SSDP) LayerType() gopacket.LayerType {
	return LayerTypeSSDP
}

func (s *SSDP) Payload() []byte {
	return nil
}

func decodeSSDP(data []byte, p gopacket.PacketBuilder) error {
	s := &SSDP{}
	err := s.DecodeFromBytes(data, p)
	if err != nil {
		return err
	}
	p.AddLayer(s)
	p.SetApplicationLayer(s)
	return nil
}

func (s *SSDP) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	if !bytes.Contains(data, []byte("HTTP/1.1")) {
		return errSSDPInvalidPacket
	}
	// since there are no further layers, the baselayer's content is
	// pointing to this layer
	s.BaseLayer = layers.BaseLayer{Contents: data[:]}

	rd := bufio.NewReader(bytes.NewReader(data))
	if bytes.Equal(data[:4], []byte("HTTP/1.1")) {
		// Response
		header, _, err := rd.ReadLine()
		responseHeaderRe, err := regexp.Compile("HTTP/1.1\\s+(\\d+)\\s+(\\w+)")
		if err != nil {
			panic(err)
		}
		matches := responseHeaderRe.FindSubmatch(header)
		s.StatusCode, err = strconv.Atoi(string(matches[1]))
		s.Status = string(matches[1])
	} else {
		header, _, err := rd.ReadLine()
		re, err := regexp.Compile("(M-SEARCH|NOTIFY)|\\w+\\s+\\w+\\s+HTTP/1.1")
		if err != nil {
			panic(err)
		}
		matches := re.FindSubmatch(header)
		s.Method = string(matches[1])

	}
	headerRe, err := regexp.Compile("(.*?):\\s+(.*)")
	if err != nil {
		panic(err)
	}
	s.Headers = make(map[string]string)
	for {
		headerLine, _, err := rd.ReadLine()
		if err != nil {
			break
		}
		if len(headerLine) == 0 {
			break
		}
		matches := headerRe.FindSubmatch(headerLine)
		if len(matches) == 3 {
			s.Headers[string(matches[1])] = string(matches[2])
		}
	}
	return nil
}

var (
	errSSDPInvalidPacket = errors.New("invalid SSDP packet")
)

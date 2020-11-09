package events

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/rs/xid"

	"github.com/google/gopacket/layers"

	"github.com/bonjourmalware/melody/internal/sessions"

	"github.com/bonjourmalware/melody/internal/config"

	"github.com/bonjourmalware/melody/internal/parsing"
	"github.com/google/gopacket"
)

type HTTPEvent struct {
	Verb          string            `json:"verb"`
	Proto         string            `json:"proto"`
	RequestURI    string            `json:"URI"`
	SourcePort    uint16            `json:"src_port"`
	DestHost      string            `json:"dst_host"`
	DestPort      uint16            `json:"dst_port"`
	Headers       map[string]string `json:"headers"`
	HeadersKeys   []string          `json:"headers_keys"`
	HeadersValues []string          `json:"headers_values"`
	InlineHeaders []string
	Errors        []string `json:"errors"`
	Body          Payload  `json:"body"`
	IsTLS         bool     `json:"is_tls"`
	Req           *http.Request
	LogData       HTTPEventLog
	BaseEvent
}

func (ev HTTPEvent) GetIPHeader() *layers.IPv4 {
	return nil
}

func (ev HTTPEvent) GetHTTPData() HTTPEvent {
	return ev
}

func (ev HTTPEvent) ToLog() EventLog {
	ev.LogData = HTTPEventLog{}
	ev.LogData.Timestamp = time.Now().Format(time.RFC3339Nano)
	//ev.LogData.NsTimestamp = strconv.FormatInt(time.Now().UnixNano(), 10)
	ev.LogData.Type = ev.Kind
	ev.LogData.SourceIP = ev.SourceIP
	ev.LogData.DestPort = ev.DestPort
	ev.LogData.Session = ev.Session

	// Deduplicate tags
	if len(ev.Tags) == 0 {
		ev.LogData.Tags = []string{}
	} else {
		var set = make(map[string]struct{})
		for _, tag := range ev.Tags {
			if _, ok := set[tag]; !ok {
				set[tag] = struct{}{}
			}
		}

		for tag := range set {
			ev.LogData.Tags = append(ev.LogData.Tags, tag)
		}
	}

	ev.LogData.Session = ev.Session
	ev.LogData.HTTP.Verb = ev.Verb
	ev.LogData.HTTP.Proto = ev.Proto
	ev.LogData.HTTP.RequestURI = ev.RequestURI
	ev.LogData.HTTP.SourcePort = ev.SourcePort
	ev.LogData.HTTP.DestHost = ev.DestHost
	ev.LogData.DestPort = ev.DestPort
	ev.LogData.SourceIP = ev.SourceIP
	ev.LogData.HTTP.Headers = ev.Headers
	ev.LogData.HTTP.Body = ev.Body
	ev.LogData.HTTP.IsTLS = ev.IsTLS
	ev.LogData.Additional = ev.Additional

	if val, ok := ev.Headers["User-Agent"]; ok {
		ev.LogData.HTTP.UserAgent = val
	}

	var headersKeys []string
	var headersValues []string

	for key, val := range ev.Headers {
		headersKeys = append(headersKeys, key)
		headersValues = append(headersValues, val)
	}

	ev.LogData.HTTP.HeadersKeys = headersKeys
	ev.LogData.HTTP.HeadersValues = headersValues

	return ev.LogData
}

func NewHTTPEvent(r *http.Request, network gopacket.Flow, transport gopacket.Flow) (*HTTPEvent, error) {
	headers := make(map[string]string)
	var inlineHeaders []string
	var errs []string
	var params []byte
	var err error

	for header := range r.Header {
		headers[header] = r.Header.Get(header)
		inlineHeaders = append(inlineHeaders, header+": "+r.Header.Get(header))
	}

	dstPort, _ := strconv.ParseUint(transport.Dst().String(), 10, 16)
	srcPort, _ := strconv.ParseUint(transport.Src().String(), 10, 16)

	params, err = parsing.GetBodyPayload(r)
	if err != nil {
		errs = append(errs, err.Error())
	}

	ev := &HTTPEvent{
		Verb:          r.Method,
		Proto:         r.Proto,
		RequestURI:    r.URL.RequestURI(),
		SourcePort:    uint16(srcPort),
		DestPort:      uint16(dstPort),
		DestHost:      network.Dst().String(),
		Body:          NewPayload(params, config.Cfg.MaxPOSTDataSize),
		IsTLS:         r.TLS != nil,
		Headers:       headers,
		InlineHeaders: inlineHeaders,
		Errors:        errs,
	}

	// Cannot use promoted (inherited) fields in struct literal
	ev.Session = sessions.Map.GetUID(transport.String())
	ev.SourceIP = network.Src().String()
	ev.Tags = []string{}
	ev.Additional = make(map[string]string)

	if ev.IsTLS {
		ev.Kind = config.HTTPSKind
	} else {
		ev.Kind = config.HTTPKind
	}

	return ev, nil
}

func NewHTTPEventFromRequest(r *http.Request) (*HTTPEvent, error) {
	headers := make(map[string]string)
	var inlineHeaders []string
	var errs []string
	var params []byte
	var srcIP string
	var dstHost string
	var rawDstPort string
	var rawSrcPort string
	var err error

	for header := range r.Header {
		headers[header] = r.Header.Get(header)
		inlineHeaders = append(inlineHeaders, header+": "+r.Header.Get(header))
	}

	dstHost, rawDstPort, err = net.SplitHostPort(r.Host)
	if err != nil {
		errs = append(errs, err.Error())
	}

	srcIP, rawSrcPort, err = net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		errs = append(errs, err.Error())
	}

	params, err = parsing.GetBodyPayload(r)
	if err != nil {
		errs = append(errs, err.Error())
	}

	srcPort, _ := strconv.ParseUint(rawSrcPort, 10, 16)
	dstPort, _ := strconv.ParseUint(rawDstPort, 10, 16)

	ev := &HTTPEvent{
		Verb:          r.Method,
		Proto:         r.Proto,
		RequestURI:    r.URL.RequestURI(),
		SourcePort:    uint16(srcPort),
		DestPort:      uint16(dstPort),
		DestHost:      dstHost,
		Body:          NewPayload(params, config.Cfg.MaxPOSTDataSize),
		IsTLS:         r.TLS != nil,
		Headers:       headers,
		InlineHeaders: inlineHeaders,
		Errors:        errs,
	}

	// Cannot use promoted (inherited) fields in struct literal
	ev.Session = xid.New().String()
	ev.SourceIP = srcIP
	ev.Tags = []string{}
	ev.Additional = make(map[string]string)

	if ev.IsTLS {
		ev.Kind = config.HTTPSKind
	} else {
		ev.Kind = config.HTTPKind
	}

	return ev, nil
}

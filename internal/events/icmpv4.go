package events

import (
	"time"

	"github.com/bonjourmalware/melody/internal/events/helpers"
	"github.com/bonjourmalware/melody/internal/events/logdata"

	"github.com/bonjourmalware/melody/internal/config"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// ICMPv4Event describes the structure of an event generated by an ICPMv4 packet
type ICMPv4Event struct {
	//ICMPv4Header *layers.ICMPv4
	LogData logdata.ICMPv4EventLog
	BaseEvent
	helpers.IPv4Layer
	helpers.ICMPv4Layer
}

// NewICMPv4Event created a new ICMPv4Event from a packet
func NewICMPv4Event(packet gopacket.Packet) (*ICMPv4Event, error) {
	var ev = &ICMPv4Event{}
	ev.Kind = config.ICMPv4Kind

	ev.Session = "n/a"
	ev.Timestamp = packet.Metadata().Timestamp

	ICMPv4Header, _ := packet.Layer(layers.LayerTypeICMPv4).(*layers.ICMPv4)
	ev.ICMPv4Layer = helpers.ICMPv4Layer{Header: ICMPv4Header}

	IPHeader, _ := packet.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	ev.IPv4Layer = helpers.IPv4Layer{Header: IPHeader}
	ev.SourceIP = ev.IPv4Layer.Header.SrcIP.String()
	ev.Additional = make(map[string]string)
	ev.Tags = make(Tags)

	return ev, nil
}

// ToLog parses the event structure and generate an EventLog almost ready to be sent to the logging file
func (ev ICMPv4Event) ToLog() EventLog {
	ev.LogData = logdata.ICMPv4EventLog{}
	ev.LogData.Timestamp = ev.Timestamp.Format(time.RFC3339Nano)

	ev.LogData.Type = ev.Kind
	ev.LogData.SourceIP = ev.SourceIP
	ev.LogData.DestPort = ev.DestPort
	ev.LogData.Session = ev.Session

	if len(ev.Tags) == 0 {
		ev.LogData.Tags = make(map[string][]string)
	} else {
		ev.LogData.Tags = ev.Tags
	}

	ev.LogData.ICMPv4 = logdata.ICMPv4LogData{
		TypeCode:     ev.ICMPv4Layer.Header.TypeCode,
		Type:         ev.ICMPv4Layer.Header.TypeCode.Type(),
		Code:         ev.ICMPv4Layer.Header.TypeCode.Code(),
		TypeCodeName: ev.ICMPv4Layer.Header.TypeCode.String(),
		Checksum:     ev.ICMPv4Layer.Header.Checksum,
		ID:           ev.ICMPv4Layer.Header.Id,
		Seq:          ev.ICMPv4Layer.Header.Seq,
		Payload:      logdata.NewPayloadLogData(ev.ICMPv4Layer.Header.Payload, config.Cfg.MaxICMPv4DataSize),
	}

	ev.LogData.IP = logdata.NewIPv4LogData(ev.IPv4Layer)
	ev.LogData.Additional = ev.Additional

	return ev.LogData
}

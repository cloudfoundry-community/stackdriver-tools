package cloudfoundry

import (
	"encoding/binary"
	"strconv"

	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
)

// Converts a v1 proto (Sonde via Firehose) to a v2 proto (gRPC via RLP)
// Conversion based on mapping at https://github.com/cloudfoundry/loggregator-api#v2---v1-mapping
func ToV1(v2 *loggregator_v2.Envelope) (*events.Envelope, error) {
	// Base envelope
	v1 := &events.Envelope{
		Timestamp:  &v2.Timestamp,
		Tags:       v2.Tags,
		Origin:     sptr(v2.GetTags()["origin"]),
		Deployment: sptr(v2.GetTags()["deployment"]),
		Job:        sptr(v2.GetTags()["job"]),
		Index:      sptr(v2.GetTags()["index"]),
		Ip:         sptr(v2.GetTags()["ip"]),
	}

	// HttpStartStop
	if timer := v2.GetTimer(); timer != nil {
		v1.HttpStartStop = &events.HttpStartStop{
			StartTimestamp: &timer.Start,
			StopTimestamp:  &timer.Stop,
			ApplicationId:  uuid(v2.GetSourceId()),
			RequestId:      uuid(v2.GetTags()["request_id"]),
			PeerType:       peertype(v2.GetTags()["peer_type"]),
			Method:         method(v2.GetTags()["method"]),
			Uri:            sptr(v2.GetTags()["uri"]),
			RemoteAddress:  sptr(v2.GetTags()["remote_address"]),
			UserAgent:      sptr(v2.GetTags()["user_agent"]),
			StatusCode:     sint32p(v2.GetTags()["status_code"]),
			ContentLength:  sint64p(v2.GetTags()["content_length"]),
			InstanceIndex:  sint32p(v2.GetTags()["instance_index"]),
			//TODO(evanbrown): How do we convert the 'forwarded' tag to a slice?
			//Forwarded:      ??
		}
		return v1, nil
	}

	// LogMessage
	if log := v2.GetLog(); log != nil {
		v1.LogMessage = &events.LogMessage{
			Message: log.Payload,
			//TODO(evanbrown): derive valid log type
			MessageType:    logmessagetype(log.String()),
			Timestamp:      int64p(v2.Timestamp),
			AppId:          sptr(v2.SourceId),
			SourceType:     sptr(v2.GetTags()["source_type"]),
			SourceInstance: sptr(v2.InstanceId),
		}
		return v1, nil
	}

	// CounterEvent
	if counterEvent := v2.GetCounter(); counterEvent != nil {
		v1.CounterEvent = &events.CounterEvent{
			Name:  sptr(counterEvent.Name),
			Delta: uint64p(counterEvent.Delta),
			Total: uint64p(counterEvent.Total),
		}
		return v1, nil
	}

	// ContainerMetric
	// Requires a special check as the v2.Gauge type can represent ContainerMetric or ValueMetric
	if containerMetric := v2.GetGauge(); containerMetric != nil && isContainerMetric(v2) {
		v1.ContainerMetric = &events.ContainerMetric{
			ApplicationId:    sptr(v2.GetSourceId()),
			InstanceIndex:    sint32p(v2.GetTags()["instance_index"]),
			CpuPercentage:    float64p(containerMetric.GetMetrics()["cpu"].Value),
			MemoryBytes:      f64touint64p(containerMetric.GetMetrics()["memory"].Value),
			DiskBytes:        f64touint64p(containerMetric.GetMetrics()["disk"].Value),
			MemoryBytesQuota: f64touint64p(containerMetric.GetMetrics()["memory_quota"].Value),
			DiskBytesQuota:   f64touint64p(containerMetric.GetMetrics()["disk_quota"].Value),
		}
		return v1, nil
	}

	// ValueMetric
	if valueMetric := v2.GetGauge(); valueMetric != nil {
		v1.ValueMetric = &events.ValueMetric{
			Name:  new(string),
			Value: float64p(0.0),
			Unit:  new(string),
		}
		return v1, nil
	}
	return nil, nil
}

func isContainerMetric(e *loggregator_v2.Envelope) bool {
	g := e.GetGauge()

	// Must be a gauge
	if g == nil {
		return false
	}

	// Required gauge values in a Container Metric
	musts := []string{"cpu", "memory", "disk", "memory_quota", "disk_quota"}

	for _, v := range musts {
		if _, ok := g.GetMetrics()[v]; !ok {
			return false
		}
	}

	return true
}

func sptr(s string) *string {
	return &s
}

func peertype(s string) *events.PeerType {
	p := events.PeerType_value[s]
	pt := events.PeerType(p)
	return &pt
}

func method(s string) *events.Method {
	m := events.Method_value[s]
	mt := events.Method(m)
	return &mt
}

func logmessagetype(s string) *events.LogMessage_MessageType {
	l := events.LogMessage_MessageType_value[s]
	lt := events.LogMessage_MessageType(l)
	return &lt
}

func sint32p(s string) *int32 {
	v, _ := strconv.Atoi(s)
	v2 := int32(v)
	return &v2
}

func sint64p(s string) *int64 {
	v, _ := strconv.Atoi(s)
	v2 := int64(v)
	return &v2
}

func int32p(i int32) *int32 {
	return &i
}

func int64p(i int64) *int64 {
	return &i
}

func uint32p(i uint32) *uint32 {
	return &i
}

func uint64p(i uint64) *uint64 {
	return &i
}

func float64p(f float64) *float64 {
	return &f
}

func f64touint64p(f float64) *uint64 {
	i := uint64(f)
	return &i
}
func uuid(s string) *events.UUID {
	return &events.UUID{
		Low:  proto.Uint64(binary.LittleEndian.Uint64([]byte(s)[:8])),
		High: proto.Uint64(binary.LittleEndian.Uint64([]byte(s)[8:])),
	}
}

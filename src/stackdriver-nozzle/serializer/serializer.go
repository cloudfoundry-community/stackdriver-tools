package serializer

import (
	"fmt"

	"errors"
	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/firehose-to-syslog/utils"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/sonde-go/events"
)

type Metric struct {
	Name   string
	Value  float64
	Labels map[string]string
}

type Log struct {
	Payload interface{}
	Labels  map[string]string
}

type Serializer interface {
	GetLog(*events.Envelope) *Log
	GetMetrics(*events.Envelope) ([]*Metric, error)
	IsLog(*events.Envelope) bool
}

type cachingClientSerializer struct {
	cachingClient caching.Caching
	logger        lager.Logger
}

func NewSerializer(cachingClient caching.Caching, logger lager.Logger) Serializer {
	if cachingClient == nil {
		logger.Fatal("nilCachingClient", errors.New("caching client cannot be nil"))
	}

	cachingClient.GetAllApp()

	return &cachingClientSerializer{cachingClient, logger}
}

func (s *cachingClientSerializer) GetLog(e *events.Envelope) *Log {
	return &Log{Payload: e, Labels: s.buildLabels(e)}
}

func (s *cachingClientSerializer) GetMetrics(envelope *events.Envelope) ([]*Metric, error) {
	labels := s.buildLabels(envelope)
	switch envelope.GetEventType() {
	case events.Envelope_ValueMetric:
		valueMetric := envelope.GetValueMetric()
		return []*Metric{{
			Name:   valueMetric.GetName(),
			Value:  valueMetric.GetValue(),
			Labels: labels}}, nil
	case events.Envelope_ContainerMetric:
		containerMetric := envelope.GetContainerMetric()
		return []*Metric{
			{"diskBytesQuota", float64(containerMetric.GetDiskBytesQuota()), labels},
			{"instanceIndex", float64(containerMetric.GetInstanceIndex()), labels},
			{"cpuPercentage", float64(containerMetric.GetCpuPercentage()), labels},
			{"diskBytes", float64(containerMetric.GetDiskBytes()), labels},
			{"memoryBytes", float64(containerMetric.GetMemoryBytes()), labels},
			{"memoryBytesQuota", float64(containerMetric.GetMemoryBytesQuota()), labels},
		}, nil
	case events.Envelope_CounterEvent:
		counterEvent := envelope.GetCounterEvent()
		return []*Metric{{
			Name:   counterEvent.GetName(),
			Value:  float64(counterEvent.GetTotal()),
			Labels: labels,
		}}, nil
	default:
		return nil, fmt.Errorf("unknown event type: %v", envelope.EventType)
	}

}

func (s *cachingClientSerializer) IsLog(envelope *events.Envelope) bool {
	switch *envelope.EventType {
	case events.Envelope_ValueMetric, events.Envelope_ContainerMetric, events.Envelope_CounterEvent:
		return false
	default:
		return true
	}
}

func getApplicationId(envelope *events.Envelope) string {
	if envelope.GetEventType() == events.Envelope_HttpStartStop {
		return utils.FormatUUID(envelope.GetHttpStartStop().GetApplicationId())
	} else if envelope.GetEventType() == events.Envelope_LogMessage {
		return envelope.GetLogMessage().GetAppId()
	} else if envelope.GetEventType() == events.Envelope_ContainerMetric {
		return envelope.GetContainerMetric().GetApplicationId()
	} else {
		return ""
	}
}

func (s *cachingClientSerializer) buildLabels(envelope *events.Envelope) map[string]string {
	labels := map[string]string{}

	if envelope.Origin != nil {
		labels["origin"] = envelope.GetOrigin()
	}

	if envelope.EventType != nil {
		labels["eventType"] = envelope.GetEventType().String()
	}

	if envelope.Deployment != nil {
		labels["deployment"] = envelope.GetDeployment()
	}

	if envelope.Job != nil {
		labels["job"] = envelope.GetJob()
	}

	if envelope.Index != nil {
		labels["index"] = envelope.GetIndex()
	}

	if envelope.Ip != nil {
		labels["ip"] = envelope.GetIp()
	}

	if appId := getApplicationId(envelope); appId != "" {
		labels["applicationId"] = appId
		s.buildAppMetadataLabels(appId, labels, envelope)
	}

	return labels
}

func (s *cachingClientSerializer) buildAppMetadataLabels(appId string, labels map[string]string, envelope *events.Envelope) {
	app := s.cachingClient.GetAppInfo(appId)

	if app.Name != "" {
		labels["appName"] = app.Name
	}

	if app.SpaceName != "" {
		labels["spaceName"] = app.SpaceName
	}

	if app.SpaceGuid != "" {
		labels["spaceGuid"] = app.SpaceGuid
	}

	if app.OrgName != "" {
		labels["orgName"] = app.OrgName
	}

	if app.OrgGuid != "" {
		labels["orgGuid"] = app.OrgGuid
	}
}

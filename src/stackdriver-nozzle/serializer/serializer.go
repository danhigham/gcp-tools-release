package serializer

import (
	"fmt"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/firehose-to-syslog/utils"
	"github.com/cloudfoundry/sonde-go/events"
)

const LabelPrefix = "cloudFoundry/"

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
	GetMetrics(*events.Envelope) []*Metric
	IsLog(*events.Envelope) bool
}

type cachingClientSerializer struct {
	cachingClient caching.Caching
}

func NewSerializer(cachingClient caching.Caching) Serializer {
	return &cachingClientSerializer{cachingClient}
}

func (s *cachingClientSerializer) GetLog(e *events.Envelope) *Log {
	return &Log{Payload: e, Labels: s.buildLabels(e)}
}

func (s *cachingClientSerializer) GetMetrics(envelope *events.Envelope) []*Metric {
	switch envelope.GetEventType() {
	case events.Envelope_ValueMetric:
		return []*Metric{{
			Name:   envelope.GetValueMetric().GetName(),
			Value:  envelope.GetValueMetric().GetValue(),
			Labels: s.buildLabels(envelope)}}
	case events.Envelope_ContainerMetric:
		containerMetric := envelope.GetContainerMetric()
		labels := s.buildLabels(envelope)
		return []*Metric{
			{"diskBytesQuota", float64(containerMetric.GetDiskBytesQuota()), labels},
			{"instanceIndex", float64(containerMetric.GetInstanceIndex()), labels},
			{"cpuPercentage", float64(containerMetric.GetCpuPercentage()), labels},
			{"diskBytes", float64(containerMetric.GetDiskBytes()), labels},
			{"memoryBytes", float64(containerMetric.GetMemoryBytes()), labels},
			{"memoryBytesQuota", float64(containerMetric.GetMemoryBytesQuota()), labels},
		}
	default:
		panic(fmt.Errorf("Unknown event type: %v", envelope.EventType))
	}

}

func (s *cachingClientSerializer) IsLog(e *events.Envelope) bool {
	switch *e.EventType {
	case events.Envelope_HttpStartStop, events.Envelope_LogMessage, events.Envelope_Error:
		return true
	case events.Envelope_ValueMetric, events.Envelope_ContainerMetric:
		return false
	case events.Envelope_CounterEvent:
		//Not yet implemented as a metric
		return true
	default:
		panic(fmt.Errorf("Unknown event type: %v", e.EventType))
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
		labels[LabelPrefix+"origin"] = envelope.GetOrigin()
	}

	if envelope.EventType != nil {
		labels[LabelPrefix+"eventType"] = envelope.GetEventType().String()
	}

	if envelope.Deployment != nil {
		labels[LabelPrefix+"deployment"] = envelope.GetDeployment()
	}

	if envelope.Job != nil {
		labels[LabelPrefix+"job"] = envelope.GetJob()
	}

	if envelope.Index != nil {
		labels[LabelPrefix+"index"] = envelope.GetIndex()
	}

	if envelope.Ip != nil {
		labels[LabelPrefix+"ip"] = envelope.GetIp()
	}

	if appId := getApplicationId(envelope); appId != "" {
		labels[LabelPrefix+"applicationId"] = appId
		s.buildAppMetadataLabels(appId, labels, envelope)
	}

	return labels
}

func (s *cachingClientSerializer) buildAppMetadataLabels(appId string, labels map[string]string, envelope *events.Envelope) {
	if s.cachingClient == nil {
		return
	}

	app := s.cachingClient.GetAppInfo(appId)

	if app.Name != "" {
		labels[LabelPrefix+"appName"] = app.Name
	}

	if app.SpaceName != "" {
		labels[LabelPrefix+"spaceName"] = app.SpaceName
	}

	if app.SpaceGuid != "" {
		labels[LabelPrefix+"spaceGuid"] = app.SpaceGuid
	}

	if app.OrgName != "" {
		labels[LabelPrefix+"orgName"] = app.OrgName
	}

	if app.OrgGuid != "" {
		labels[LabelPrefix+"orgGuid"] = app.OrgGuid
	}
}

package nozzle

import (
	"fmt"
	"strings"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
)

type PostContainerMetricError struct {
	Errors []error
}

func (e *PostContainerMetricError) Error() string {
	errors := []string{}
	for _, err := range e.Errors {
		errors = append(errors, err.Error())
	}
	return strings.Join(errors, "\n")
}

type Nozzle struct {
	StackdriverClient stackdriver.Client
}

func (n *Nozzle) HandleEvent(eventsEnvelope *events.Envelope) error {
	envelope := Envelope{eventsEnvelope}
	labels := envelope.Labels()

	switch envelope.GetEventType() {
	case events.Envelope_ContainerMetric:
		return n.postContainerMetrics(envelope)
	case events.Envelope_ValueMetric:
		valueMetric := envelope.GetValueMetric()
		name := valueMetric.GetName()
		value := valueMetric.GetValue()

		err := n.StackdriverClient.PostMetric(name, value, labels)
		return err
	default:
		n.StackdriverClient.PostLog(envelope, labels)
		return nil
	}
}

func (n *Nozzle) postContainerMetrics(envelope Envelope) *PostContainerMetricError {
	containerMetric := envelope.GetContainerMetric()

	labels := envelope.Labels()
	labels["applicationId"] = containerMetric.GetApplicationId()

	errorsCh := make(chan error)

	n.postContainerMetric(errorsCh, "diskBytesQuota", float64(containerMetric.GetDiskBytesQuota()), labels)
	n.postContainerMetric(errorsCh, "instanceIndex", float64(containerMetric.GetInstanceIndex()), labels)
	n.postContainerMetric(errorsCh, "cpuPercentage", float64(containerMetric.GetCpuPercentage()), labels)
	n.postContainerMetric(errorsCh, "diskBytes", float64(containerMetric.GetDiskBytes()), labels)
	n.postContainerMetric(errorsCh, "memoryBytes", float64(containerMetric.GetMemoryBytes()), labels)
	n.postContainerMetric(errorsCh, "memoryBytesQuota", float64(containerMetric.GetMemoryBytesQuota()), labels)

	errors := []error{}
	for i := 0; i < 6; i++ {
		err := <-errorsCh
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) == 0 {
		return nil
	} else {
		return &PostContainerMetricError{
			Errors: errors,
		}
	}
}

func (n *Nozzle) postContainerMetric(errorsCh chan error, name string, value float64, labels map[string]string) {
	go func() {
		err := n.StackdriverClient.PostMetric(name, value, labels)
		if err != nil {
			errorsCh <- fmt.Errorf("%v: %v", name, err.Error())
		} else {
			errorsCh <- nil
		}
	}()
}

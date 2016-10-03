package nozzle_test

import (
	"time"

	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/nozzle"

	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Envelope", func() {
	It("has labels equivalent to its fields", func() {
		origin := "cool-origin"
		eventType := events.Envelope_HttpStartStop
		timestamp := time.Now().UnixNano()
		deployment := "neat-deployment"
		job := "some-job"
		index := "an-index"
		ip := "192.168.1.1"
		tags := map[string]string{
			"foo": "bar",
		}

		envelope := nozzle.Envelope{
			Envelope: &events.Envelope{
				Origin:     &origin,
				EventType:  &eventType,
				Timestamp:  &timestamp,
				Deployment: &deployment,
				Job:        &job,
				Index:      &index,
				Ip:         &ip,
				Tags:       tags,
			},
		}

		labels := envelope.Labels()
		Expect(labels).To(Equal(map[string]string{
			"origin": origin,
			"event_type": eventType.String(),
			"deployment": deployment,
			"job": job,
			"index": index,
			"ip": ip,
		}))
	})

	It("ignores empty fields", func() {
		origin := "cool-origin"
		eventType := events.Envelope_HttpStartStop
		timestamp := time.Now().UnixNano()
		job := "some-job"
		index := "an-index"
		tags := map[string]string{
			"foo": "bar",
		}

		envelope := nozzle.Envelope{
			Envelope: &events.Envelope{
				Origin:     &origin,
				EventType:  &eventType,
				Timestamp:  &timestamp,
				Deployment: nil,
				Job:        &job,
				Index:      &index,
				Ip:         nil,
				Tags:       tags,
			},
		}

		labels := envelope.Labels()
		Expect(labels).To(Equal(map[string]string{
			"origin": origin,
			"event_type": eventType.String(),
			"job": job,
			"index": index,
		}))
	})

})

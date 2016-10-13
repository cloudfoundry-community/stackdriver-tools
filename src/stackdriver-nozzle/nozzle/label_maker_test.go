package nozzle_test

import (
	"time"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LabelMaker", func() {
	var (
		subject  nozzle.LabelMaker
		envelope *events.Envelope
	)

	BeforeEach(func() {
		subject = nozzle.NewLabelMaker(caching.NewCachingEmpty())
	})

	It("makes labels from envelopes", func() {
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

		envelope = &events.Envelope{
			Origin:     &origin,
			EventType:  &eventType,
			Timestamp:  &timestamp,
			Deployment: &deployment,
			Job:        &job,
			Index:      &index,
			Ip:         &ip,
			Tags:       tags,
		}

		labels := subject.Build(envelope)

		Expect(labels).To(Equal(map[string]string{
			"origin":    origin,
			"eventType": eventType.String(),
			"job":       job,
			"index":     index,
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

		envelope := &events.Envelope{
			Origin:     &origin,
			EventType:  &eventType,
			Timestamp:  &timestamp,
			Deployment: nil,
			Job:        &job,
			Index:      &index,
			Ip:         nil,
			Tags:       tags,
		}

		labels := subject.Build(envelope)

		Expect(labels).To(Equal(map[string]string{
			"origin":    origin,
			"eventType": eventType.String(),
			"job":       job,
			"index":     index,
		}))
	})

	Context("Metadata", func() {
		var (
			appGuid = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
			low     = uint64(0x7243cc580bc17af4)
			high    = uint64(0x79d4c3b2020e67a5)
			appId   = events.UUID{Low: &low, High: &high}
		)

		Context("application id", func() {
			It("httpStartStop adds app id when present", func() {
				eventType := events.Envelope_HttpStartStop

				event := events.HttpStartStop{
					ApplicationId: &appId,
				}
				envelope := &events.Envelope{
					EventType:     &eventType,
					HttpStartStop: &event,
				}

				labels := subject.Build(envelope)

				Expect(labels["applicationId"]).To(Equal(appGuid))
			})

			It("LogMessage adds app id", func() {
				eventType := events.Envelope_LogMessage

				event := events.LogMessage{
					AppId: &appGuid,
				}
				envelope := &events.Envelope{
					EventType:  &eventType,
					LogMessage: &event,
				}

				labels := subject.Build(envelope)

				Expect(labels["applicationId"]).To(Equal(appGuid))
			})

			It("ValueMetric does not add app id", func() {
				eventType := events.Envelope_ValueMetric

				event := events.ValueMetric{}
				envelope := &events.Envelope{
					EventType:   &eventType,
					ValueMetric: &event,
				}

				labels := subject.Build(envelope)
				Expect(labels).NotTo(HaveKey("applicationId"))
			})

			It("CounterEvent does not add app id", func() {
				eventType := events.Envelope_CounterEvent

				event := events.CounterEvent{}
				envelope := &events.Envelope{
					EventType:    &eventType,
					CounterEvent: &event,
				}

				labels := subject.Build(envelope)

				Expect(labels).NotTo(HaveKey("applicationId"))
			})

			It("Error does not add app id", func() {
				eventType := events.Envelope_Error

				event := events.Error{}
				envelope := &events.Envelope{
					EventType: &eventType,
					Error:     &event,
				}

				labels := subject.Build(envelope)

				Expect(labels).NotTo(HaveKey("applicationId"))
			})

			It("ContainerMetric does add app id", func() {
				eventType := events.Envelope_ContainerMetric

				event := events.ContainerMetric{
					ApplicationId: &appGuid,
				}
				envelope := &events.Envelope{
					EventType:       &eventType,
					ContainerMetric: &event,
				}

				labels := subject.Build(envelope)

				Expect(labels["applicationId"]).To(Equal(appGuid))
			})
		})

		Context("application metadata", func() {
			var (
				cachingClient *mocks.CachingClient
			)

			BeforeEach(func() {
				cachingClient = &mocks.CachingClient{}
				cachingClient.AppInfo = make(map[string]caching.App)

				subject = nozzle.NewLabelMaker(cachingClient)
			})

			Context("for a LogMessage", func() {
				var (
					eventType = events.Envelope_LogMessage
					event     *events.LogMessage
					envelope  *events.Envelope
					spaceGuid = "2ab560c3-3f21-45e0-9452-d748ff3a15e9"
					orgGuid   = "b494fb47-3c44-4a98-9a08-d839ec5c799b"
				)

				BeforeEach(func() {
					event = &events.LogMessage{
						AppId: &appGuid,
					}
					envelope = &events.Envelope{
						EventType:  &eventType,
						LogMessage: event,
					}
				})

				It("adds fields for a resolved app", func() {
					app := caching.App{
						Name:      "MyApp",
						Guid:      appGuid,
						SpaceName: "MySpace",
						SpaceGuid: spaceGuid,
						OrgName:   "MyOrg",
						OrgGuid:   orgGuid,
					}

					cachingClient.AppInfo[appGuid] = app

					labels := subject.Build(envelope)

					Expect(labels).To(HaveKeyWithValue("appName", app.Name))
					Expect(labels).To(HaveKeyWithValue("spaceName", app.SpaceName))
					Expect(labels).To(HaveKeyWithValue("spaceGuid", app.SpaceGuid))
					Expect(labels).To(HaveKeyWithValue("orgName", app.OrgName))
					Expect(labels).To(HaveKeyWithValue("orgGuid", app.OrgGuid))
				})

				It("doesn't add fields for an unresolved app", func() {
					labels := subject.Build(envelope)

					Expect(labels).NotTo(HaveKey("appName"))
					Expect(labels).NotTo(HaveKey("spaceName"))
					Expect(labels).NotTo(HaveKey("spaceGuid"))
					Expect(labels).NotTo(HaveKey("orgName"))
					Expect(labels).NotTo(HaveKey("orgGuid"))
				})
			})
		})
	})
})

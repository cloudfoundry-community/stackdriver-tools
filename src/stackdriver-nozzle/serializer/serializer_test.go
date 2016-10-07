package serializer_test

import (
	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry/sonde-go/events"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"stackdriver-nozzle/mocks"
	"stackdriver-nozzle/serializer"
)

var _ = Describe("Serializer", func() {
	var (
		subject serializer.Serializer
		logger  *mocks.MockLogger
	)

	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		subject = serializer.NewSerializer(caching.NewCachingEmpty(), logger)
	})

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

		envelope := &events.Envelope{
			Origin:     &origin,
			EventType:  &eventType,
			Timestamp:  &timestamp,
			Deployment: &deployment,
			Job:        &job,
			Index:      &index,
			Ip:         &ip,
			Tags:       tags,
		}

		log := subject.GetLog(envelope)

		labels := log.Labels
		Expect(labels).To(Equal(map[string]string{
			"origin":     origin,
			"eventType":  eventType.String(),
			"deployment": deployment,
			"job":        job,
			"index":      index,
			"ip":         ip,
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

		log := subject.GetLog(envelope)
		labels := log.Labels

		Expect(labels).To(Equal(map[string]string{
			"origin":    origin,
			"eventType": eventType.String(),
			"job":       job,
			"index":     index,
		}))
	})

	Context("GetMetrics", func() {
		It("creates the proper metrics for ContainerMetric", func() {
			diskBytesQuota := uint64(1073741824)
			instanceIndex := int32(0)
			cpuPercentage := 0.061651273460637
			diskBytes := uint64(164634624)
			memoryBytes := uint64(16601088)
			memoryBytesQuota := uint64(33554432)
			applicationId := "ee2aa52e-3c8a-4851-b505-0cb9fe24806e"

			metricType := events.Envelope_ContainerMetric
			containerMetric := events.ContainerMetric{
				DiskBytesQuota:   &diskBytesQuota,
				InstanceIndex:    &instanceIndex,
				CpuPercentage:    &cpuPercentage,
				DiskBytes:        &diskBytes,
				MemoryBytes:      &memoryBytes,
				MemoryBytesQuota: &memoryBytesQuota,
				ApplicationId:    &applicationId,
			}

			envelope := &events.Envelope{
				EventType:       &metricType,
				ContainerMetric: &containerMetric,
			}

			labels := map[string]string{
				"eventType":     "ContainerMetric",
				"applicationId": applicationId,
			}

			metrics, err := subject.GetMetrics(envelope)
			Expect(err).To(BeNil())

			Expect(metrics).To(HaveLen(6))

			Expect(metrics).To(ContainElement(&serializer.Metric{"diskBytesQuota", float64(1073741824), labels}))
			Expect(metrics).To(ContainElement(&serializer.Metric{"instanceIndex", float64(0), labels}))
			Expect(metrics).To(ContainElement(&serializer.Metric{"cpuPercentage", 0.061651273460637, labels}))
			Expect(metrics).To(ContainElement(&serializer.Metric{"diskBytes", float64(164634624), labels}))
			Expect(metrics).To(ContainElement(&serializer.Metric{"memoryBytes", float64(16601088), labels}))
			Expect(metrics).To(ContainElement(&serializer.Metric{"memoryBytesQuota", float64(33554432), labels}))
		})

		It("creates metric for CounterEvent", func() {
			eventType := events.Envelope_CounterEvent
			name := "counterName"
			total := uint64(123456)

			event := events.CounterEvent{
				Name:  &name,
				Total: &total,
			}
			envelope := &events.Envelope{
				EventType:    &eventType,
				CounterEvent: &event,
			}

			labels := map[string]string{
				"eventType": "CounterEvent",
			}

			metrics, err := subject.GetMetrics(envelope)
			Expect(err).To(BeNil())
			Expect(metrics).To(HaveLen(1))
			Expect(metrics).To(ContainElement(&serializer.Metric{"counterName", float64(123456), labels}))
		})

		It("returns error when envelope contains unhandled event type", func() {
			eventType := events.Envelope_HttpStart
			envelope := &events.Envelope{
				EventType: &eventType,
			}
			_, err := subject.GetMetrics(envelope)
			Expect(err).NotTo(BeNil())
		})
	})

	Context("isLog", func() {
		It("HttpStartStop is log", func() {
			eventType := events.Envelope_HttpStartStop

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeTrue())
		})

		It("LogMessage is log", func() {
			eventType := events.Envelope_LogMessage

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeTrue())
		})

		It("ValueMetric is *NOT* log", func() {
			eventType := events.Envelope_ValueMetric

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeFalse())
		})

		It("CounterEvent is *NOT* log", func() {
			eventType := events.Envelope_CounterEvent

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeFalse())

		})

		It("Error is log", func() {
			eventType := events.Envelope_Error

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeTrue())

		})

		It("ContainerMetric is *NOT* log", func() {
			eventType := events.Envelope_ContainerMetric

			envelope := &events.Envelope{
				EventType: &eventType,
			}
			Expect(subject.IsLog(envelope)).To(BeFalse())

		})

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

				log := subject.GetLog(envelope)
				labels := log.Labels

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

				log := subject.GetLog(envelope)
				labels := log.Labels
				Expect(labels["applicationId"]).To(Equal(appGuid))

			})

			It("ValueMetric does not add app id", func() {
				eventType := events.Envelope_ValueMetric

				event := events.ValueMetric{}
				envelope := &events.Envelope{
					EventType:   &eventType,
					ValueMetric: &event,
				}
				metrics, err := subject.GetMetrics(envelope)
				Expect(err).To(BeNil())

				Expect(metrics).To(HaveLen(1))
				valueMetric := metrics[0]

				labels := valueMetric.Labels
				Expect(labels).NotTo(HaveKey("applicationId"))

			})

			It("CounterEvent does not add app id", func() {
				eventType := events.Envelope_CounterEvent

				event := events.CounterEvent{}
				envelope := &events.Envelope{
					EventType:    &eventType,
					CounterEvent: &event,
				}
				metrics, err := subject.GetMetrics(envelope)
				Expect(err).To(BeNil())

				Expect(metrics).To(HaveLen(1))
				valueMetric := metrics[0]

				labels := valueMetric.Labels
				Expect(labels).NotTo(HaveKey("applicationId"))
			})

			It("Error does not add app id", func() {
				eventType := events.Envelope_Error

				event := events.Error{}
				envelope := &events.Envelope{
					EventType: &eventType,
					Error:     &event,
				}

				log := subject.GetLog(envelope)
				labels := log.Labels
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

				metrics, err := subject.GetMetrics(envelope)

				Expect(err).To(BeNil())

				Expect(len(metrics)).To(Not(Equal(0)))

				for _, metric := range metrics {
					labels := metric.Labels
					Expect(labels["applicationId"]).To(Equal(appGuid))

				}
			})
		})

		Context("application metadata", func() {
			var (
				cachingClient MockCachingClient
			)

			BeforeEach(func() {
				cachingClient = MockCachingClient{}
				cachingClient.AppInfo = make(map[string]caching.App)
				subject = serializer.NewSerializer(&cachingClient, nil)
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

					log := subject.GetLog(envelope)
					labels := log.Labels

					Expect(labels).To(HaveKeyWithValue("appName", app.Name))
					Expect(labels).To(HaveKeyWithValue("spaceName", app.SpaceName))
					Expect(labels).To(HaveKeyWithValue("spaceGuid", app.SpaceGuid))
					Expect(labels).To(HaveKeyWithValue("orgName", app.OrgName))
					Expect(labels).To(HaveKeyWithValue("orgGuid", app.OrgGuid))
				})

				It("doesn't add fields for an unresolved app", func() {
					log := subject.GetLog(envelope)
					labels := log.Labels

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

type MockCachingClient struct {
	AppInfo map[string]caching.App
}

func (c *MockCachingClient) CreateBucket() {
	panic("unexpected")
}

func (c *MockCachingClient) PerformPoollingCaching(tickerTime time.Duration) {
	panic("unexpected")
}

func (c *MockCachingClient) fillDatabase(listApps []caching.App) {
	panic("unexpected")
}

func (c *MockCachingClient) GetAppByGuid(appGuid string) []caching.App {
	panic("unexpected")
}

func (c *MockCachingClient) GetAllApp() []caching.App {
	return []caching.App{}
}

func (c *MockCachingClient) GetAppInfo(appGuid string) caching.App {
	return c.AppInfo[appGuid]
}

func (c *MockCachingClient) Close() {
	panic("unexpected")
}

func (c *MockCachingClient) GetAppInfoCache(appGuid string) caching.App {
	panic("unexpected")
}

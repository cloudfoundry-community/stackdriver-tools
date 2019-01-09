/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nozzle

import (
	"strconv"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/telemetry"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/golang/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testApp struct {
	name      string
	low, high uint64
}

func (ta testApp) GUID() string {
	return formatUUID(ta.UUID())
}

func (ta testApp) UUID() *events.UUID {
	return &events.UUID{Low: &ta.low, High: &ta.high}
}

func (ta testApp) Index() string {
	return formatUUID(&events.UUID{Low: &ta.high, High: &ta.low})
}

func (ta testApp) AppInfo() cloudfoundry.AppInfo {
	return cloudfoundry.AppInfo{
		AppName:   ta.name,
		SpaceName: "Space",
		OrgName:   "Org",
	}
}

func (ta testApp) Events(count int, code, instanceIndex int32) []*events.Envelope {
	ret := make([]*events.Envelope, count)
	for i := 0; i < count; i++ {
		ret[i] = &events.Envelope{
			Origin:    proto.String("origin"),
			EventType: events.Envelope_HttpStartStop.Enum(),
			Job:       proto.String("router"),
			Index:     proto.String(ta.Index()),
			HttpStartStop: &events.HttpStartStop{
				StatusCode:    &code,
				InstanceIndex: &instanceIndex,
				ApplicationId: ta.UUID(),
			},
		}
	}
	return ret
}

func (ta testApp) RequestCount(instanceIndex int) int {
	key := messages.Flatten(map[string]string{
		"job":             "router",
		"index":           ta.Index(),
		"applicationPath": makePath(ta.AppInfo()),
		"instanceIndex":   strconv.Itoa(instanceIndex),
	})
	if ctr, ok := requestCount.Get(key).(*telemetry.Counter); ok {
		return int(ctr.Value())
	}
	// Zero is a perfectly valid counter value, -1 not so much.
	// Returning a separate error here is inconvenient.
	return -1
}

func (ta testApp) ResponseCode(code, instanceIndex int) int {
	key := messages.Flatten(map[string]string{
		"job":             "router",
		"index":           ta.Index(),
		"applicationPath": makePath(ta.AppInfo()),
		"instanceIndex":   strconv.Itoa(instanceIndex),
		"code":            strconv.Itoa(code),
	})
	if ctr, ok := responseCode.Get(key).(*telemetry.Counter); ok {
		return int(ctr.Value())
	}
	// Zero is a perfectly valid counter value, -1 not so much.
	// Returning a separate error here is inconvenient.
	return -1
}

var testApps = []testApp{
	{"AppOne", 0x2234878713489723, 0x9df2ba7314302765},
	{"AppTwo", 0x14338a4961390dea, 0x657381f23cc11739},
	{"AppTri", 0xf3874ab3b321bb95, 0xa285f76e81964556},
}

var _ = Describe("HttpSink", func() {
	var (
		subject    Sink
		labelMaker LabelMaker
		foundation = "cf"
		air        = &mocks.AppInfoRepository{AppInfoMap: map[string]cloudfoundry.AppInfo{}}
	)

	BeforeEach(func() {
		for _, app := range testApps {
			air.AppInfoMap[app.GUID()] = app.AppInfo()
		}
		labelMaker = NewLabelMaker(air, foundation)
		subject = NewHttpSink(&mocks.MockLogger{}, labelMaker)
	})

	It("increments counters for reqeusts", func() {
		receive := func(es []*events.Envelope) {
			for _, e := range es {
				subject.Receive(e)
			}
		}
		// AppOne has 2 instances.
		// The first serves 20 "200 OK" responses.
		receive(testApps[0].Events(20, 200, 0))
		// The second serves 18 "200 OK" and 2 "500 ISE" responses.
		receive(testApps[0].Events(18, 200, 1))
		receive(testApps[0].Events(2, 500, 1))

		// AppTwo has one instance.
		// It serves 15 "200 OK", 3 "401 Forbidden" and 7 "404 Not Found" responses.
		receive(testApps[1].Events(15, 200, 0))
		receive(testApps[1].Events(3, 401, 0))
		receive(testApps[1].Events(7, 404, 0))

		// AppTri has one instance.
		// It serves 8 302 redirects.
		receive(testApps[2].Events(8, 302, 0))

		Expect(testApps[0].RequestCount(0)).To(Equal(20))
		Expect(testApps[0].ResponseCode(200, 0)).To(Equal(20))
		Expect(testApps[0].ResponseCode(500, 0)).To(Equal(-1))

		Expect(testApps[0].RequestCount(1)).To(Equal(20))
		Expect(testApps[0].ResponseCode(200, 1)).To(Equal(18))
		Expect(testApps[0].ResponseCode(500, 1)).To(Equal(2))

		Expect(testApps[0].RequestCount(2)).To(Equal(-1))
		Expect(testApps[0].ResponseCode(200, 2)).To(Equal(-1))

		Expect(testApps[1].RequestCount(0)).To(Equal(25))
		Expect(testApps[1].ResponseCode(200, 0)).To(Equal(15))
		Expect(testApps[1].ResponseCode(401, 0)).To(Equal(3))
		Expect(testApps[1].ResponseCode(404, 0)).To(Equal(7))

		Expect(testApps[2].RequestCount(0)).To(Equal(8))
		Expect(testApps[2].ResponseCode(302, 0)).To(Equal(8))
	})
})

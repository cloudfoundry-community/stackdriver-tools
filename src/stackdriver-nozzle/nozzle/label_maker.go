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
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry/sonde-go/events"
)

type LabelMaker interface {
	MetricLabels(*events.Envelope) map[string]string
	LogLabels(*events.Envelope) map[string]string
}

func NewLabelMaker(appInfoRepository cloudfoundry.AppInfoRepository) LabelMaker {
	return &labelMaker{appInfoRepository: appInfoRepository}
}

type labelMaker struct {
	appInfoRepository cloudfoundry.AppInfoRepository
}

type labelMap map[string]string

func (labels labelMap) setIfNotEmpty(key, value string) {
	if value != "" {
		labels[key] = value
	}
}

type pathMaker struct {
	buf bytes.Buffer
}

func (pm *pathMaker) addElement(key, value string) {
	pm.buf.WriteByte('/')
	if value != "" {
		pm.buf.WriteString(value)
	} else {
		pm.buf.WriteString("unknown_")
		pm.buf.WriteString(key)
	}
}

func (pm *pathMaker) String() string {
	return pm.buf.String()
}

// MetricLabels extracts metric metadata from the event envelope and event
// contained within, and constructs a set of Stackdriver (SD) metric labels
// from them.
//
// Since SD only allows 10 custom labels per metric, we collapse application
// metadata into a "path" representing the serving application, space, and org.
// We maintain vm and application instance indexes as separate labels so that
// it is easy to aggregate across multiple instances.
func (lm *labelMaker) MetricLabels(envelope *events.Envelope) map[string]string {
	labels := labelMap{}

	labels.setIfNotEmpty("job", envelope.GetJob())
	labels.setIfNotEmpty("index", envelope.GetIndex())
	labels.setIfNotEmpty("applicationPath", lm.getApplicationPath(envelope))
	labels.setIfNotEmpty("instanceIndex", getInstanceIndex(envelope))
	labels.setIfNotEmpty("tags", getTags(envelope))

	return labels
}

// LogLabels extracts log metadata from the event envelope and event contained
// within and constructs a set of Stackdriver (SD) log labels from them.
//
// This differs from MetricLabels because we want to retain the event type
// and origin in logs so that we can process logs of a given type easily.
// The limit of 10 custom labels does not (appear to) apply to SD logging,
// so there's no risk to adding extra labels here.
func (lm *labelMaker) LogLabels(envelope *events.Envelope) map[string]string {
	labels := labelMap(lm.MetricLabels(envelope))
	labels.setIfNotEmpty("origin", envelope.GetOrigin())
	labels.setIfNotEmpty("eventType", envelope.GetEventType().String())
	return labels
}

// getApplicationPath returns a path that uniquely identifies a
// collection of instances of a given application running in an org + space.
// The path hierarchy is /org/space/application, e.g.
//     /system/autoscaling/autoscale
func (lm *labelMaker) getApplicationPath(envelope *events.Envelope) string {
	appID := getApplicationId(envelope)
	if appID == "" {
		return ""
	}
	app := lm.appInfoRepository.GetAppInfo(appID)
	if app.AppName == "" {
		return ""
	}

	path := pathMaker{}
	path.addElement("org", app.OrgName)
	path.addElement("space", app.SpaceName)
	path.addElement("application", app.AppName)

	return path.String()
}

// getApplicationId extracts the application UUID from the event contained
// within the envelope, for those events that have application IDs.
func getApplicationId(envelope *events.Envelope) string {
	switch envelope.GetEventType() {
	case events.Envelope_HttpStartStop:
		return formatUUID(envelope.GetHttpStartStop().GetApplicationId())
	case events.Envelope_LogMessage:
		return envelope.GetLogMessage().GetAppId()
	case events.Envelope_ContainerMetric:
		return envelope.GetContainerMetric().GetApplicationId()
	}
	return ""
}

// getInstanceIndex extracts the instance index or UUID from the event
// contained within the envelope, for those events that have instance IDs.
func getInstanceIndex(envelope *events.Envelope) string {
	switch envelope.GetEventType() {
	case events.Envelope_HttpStartStop:
		hss := envelope.GetHttpStartStop()
		if hss != nil && hss.InstanceIndex != nil {
			return fmt.Sprintf("%d", hss.GetInstanceIndex())
		}
		// Sometimes InstanceIndex is not set but InstanceId is; fall back.
		return hss.GetInstanceId()
	case events.Envelope_LogMessage:
		return envelope.GetLogMessage().GetSourceInstance()
	case events.Envelope_ContainerMetric:
		return fmt.Sprintf("%d", envelope.GetContainerMetric().GetInstanceIndex())
	}
	return ""
}

// getTags extracts any additional tags from the envelope and returns them
// serialized as a single comma-separated string of key=value, sorted by key.
//
// This is sub-optimal, but we can't directly map tags to labels because
// some entities writing metrics to the firehose use different tag sets
// for metrics with the same origin + name. Since metric label keys form
// part of the metric descriptor in SD, this would result in some metrics
// not having the correct descriptor and being dropped.
func getTags(envelope *events.Envelope) string {
	tags := envelope.GetTags()
	if len(tags) == 0 {
		return ""
	}

	tagKeys := make([]string, 0, len(tags))
	for k := range tags {
		tagKeys = append(tagKeys, k)
	}
	sort.Strings(tagKeys)

	tagElems := make([]string, len(tags))
	for i, k := range tagKeys {
		tagElems[i] = k + "=" + tags[k]
	}
	return strings.Join(tagElems, ",")
}

func formatUUID(uuid *events.UUID) string {
	if uuid == nil {
		return ""
	}
	var uuidBytes [16]byte
	binary.LittleEndian.PutUint64(uuidBytes[:8], uuid.GetLow())
	binary.LittleEndian.PutUint64(uuidBytes[8:], uuid.GetHigh())
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:])
}

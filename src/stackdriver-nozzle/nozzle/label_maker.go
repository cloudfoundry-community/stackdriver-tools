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
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry/sonde-go/events"
	"encoding/binary"
	"fmt"
)

type LabelMaker interface {
	Build(*events.Envelope) map[string]string
}

func NewLabelMaker(appInfoRepository cloudfoundry.AppInfoRepository) LabelMaker {
	return &labelMaker{appInfoRepository: appInfoRepository}
}

type labelMaker struct {
	appInfoRepository cloudfoundry.AppInfoRepository
}

func (lm *labelMaker) Build(envelope *events.Envelope) map[string]string {
	labels := map[string]string{}

	if envelope.Origin != nil {
		labels["origin"] = envelope.GetOrigin()
	}

	if envelope.EventType != nil {
		labels["eventType"] = envelope.GetEventType().String()
	}

	if envelope.Job != nil {
		labels["job"] = envelope.GetJob()
	}

	if envelope.Index != nil {
		labels["index"] = envelope.GetIndex()
	}

	if appId := lm.getApplicationId(envelope); appId != "" {
		labels["applicationId"] = appId
		lm.buildAppMetadataLabels(appId, labels, envelope)
	}

	return labels
}

func (lm *labelMaker) getApplicationId(envelope *events.Envelope) string {
	if envelope.GetEventType() == events.Envelope_HttpStartStop {
		return formatUUID(envelope.GetHttpStartStop().GetApplicationId())
	} else if envelope.GetEventType() == events.Envelope_LogMessage {
		return envelope.GetLogMessage().GetAppId()
	} else if envelope.GetEventType() == events.Envelope_ContainerMetric {
		return envelope.GetContainerMetric().GetApplicationId()
	} else {
		return ""
	}
}

func (lm *labelMaker) buildAppMetadataLabels(guid string, labels map[string]string, envelope *events.Envelope) {
	app := lm.appInfoRepository.GetAppInfo(guid)

	if app.AppName != "" {
		labels["appName"] = app.AppName
	}

	if app.SpaceName != "" {
		labels["spaceName"] = app.SpaceName
	}

	if app.SpaceGUID != "" {
		labels["spaceGuid"] = app.SpaceGUID
	}

	if app.OrgName != "" {
		labels["orgName"] = app.OrgName
	}

	if app.OrgGUID != "" {
		labels["orgGuid"] = app.OrgGUID
	}
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

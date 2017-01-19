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
	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
	"github.com/cloudfoundry-community/firehose-to-syslog/utils"
	"github.com/cloudfoundry/sonde-go/events"
)

type LabelMaker interface {
	Build(*events.Envelope) map[string]string
}

func NewLabelMaker(cachingClient caching.Caching) LabelMaker {
	return &labelMaker{cachingClient: cachingClient}
}

type labelMaker struct {
	cachingClient caching.Caching
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
		return utils.FormatUUID(envelope.GetHttpStartStop().GetApplicationId())
	} else if envelope.GetEventType() == events.Envelope_LogMessage {
		return envelope.GetLogMessage().GetAppId()
	} else if envelope.GetEventType() == events.Envelope_ContainerMetric {
		return envelope.GetContainerMetric().GetApplicationId()
	} else {
		return ""
	}
}

func (lm *labelMaker) buildAppMetadataLabels(appId string, labels map[string]string, envelope *events.Envelope) {
	app := lm.cachingClient.GetAppInfoCache(appId)

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

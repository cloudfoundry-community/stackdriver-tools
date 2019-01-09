/*
 * Copyright 2019 Google Inc.
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

package cloudfoundry

import "github.com/cloudfoundry-community/go-cfclient"

type AppInfoRepository interface {
	GetAppInfo(string) AppInfo
}

type AppInfo struct {
	AppName   string
	SpaceGUID string
	SpaceName string
	OrgGUID   string
	OrgName   string
}

func NewAppInfoRepository(cfClient *cfclient.Client) AppInfoRepository {
	return &appInfoRepository{cfClient, map[string]AppInfo{}}
}

func NullAppInfoRepository() AppInfoRepository {
	return &nullAppInfoRepository{}
}

type appInfoRepository struct {
	cfClient *cfclient.Client
	cache    map[string]AppInfo
}

func (air *appInfoRepository) GetAppInfo(guid string) AppInfo {
	appInfo, ok := air.cache[guid]
	if !ok {
		app, err := air.cfClient.AppByGuid(guid)
		if err == nil {
			appInfo = AppInfo{
				AppName:   app.Name,
				SpaceGUID: app.SpaceData.Entity.Guid,
				SpaceName: app.SpaceData.Entity.Name,
				OrgGUID:   app.SpaceData.Entity.OrgData.Entity.Guid,
				OrgName:   app.SpaceData.Entity.OrgData.Entity.Name,
			}
			air.cache[guid] = appInfo
		}
	}
	return appInfo
}

type nullAppInfoRepository struct{}

func (nair *nullAppInfoRepository) GetAppInfo(guid string) AppInfo {
	return AppInfo{}
}

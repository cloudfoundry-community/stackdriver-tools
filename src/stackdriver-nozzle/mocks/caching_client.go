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

package mocks

import (
	"time"

	"github.com/cloudfoundry-community/firehose-to-syslog/caching"
)

type CachingClient struct {
	AppInfo map[string]caching.App
}

func (c *CachingClient) CreateBucket() {
	panic("unexpected")
}

func (c *CachingClient) PerformPoollingCaching(tickerTime time.Duration) {
	panic("unexpected")
}

func (c *CachingClient) fillDatabase(listApps []caching.App) {
	panic("unexpected")
}

func (c *CachingClient) GetAppByGuid(appGuid string) []caching.App {
	panic("unexpected")
}

func (c *CachingClient) GetAllApp() []caching.App {
	return []caching.App{}
}

func (c *CachingClient) GetAppInfo(appGuid string) caching.App {
	panic("unexpected")
}

func (c *CachingClient) Close() {
	panic("unexpected")
}

func (c *CachingClient) GetAppInfoCache(appGuid string) caching.App {
	return c.AppInfo[appGuid]
}

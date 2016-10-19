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

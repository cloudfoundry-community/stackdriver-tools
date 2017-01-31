package mocks

import "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"

type AppInfoRepository struct {
	AppInfoMap map[string]cloudfoundry.AppInfo
}

func (air *AppInfoRepository) GetAppInfo(guid string) cloudfoundry.AppInfo {
	return air.AppInfoMap[guid]
}

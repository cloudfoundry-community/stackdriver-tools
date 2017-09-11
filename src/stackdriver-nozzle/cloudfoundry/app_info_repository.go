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
		if err != nil {
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

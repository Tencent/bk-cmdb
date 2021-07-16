/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"context"

	"configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/backbone"
	"configcenter/src/common/errors"
	"configcenter/src/common/metadata"
	"configcenter/src/common/metric"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/types"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/scene_server/admin_server/app/options"
	"configcenter/src/scene_server/admin_server/configures"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/redis"

	"github.com/emicklei/go-restful"
)

type Service struct {
	*backbone.Engine
	db           dal.RDB
	watchDB      dal.RDB
	cache        redis.Client
	ctx          context.Context
	Config       options.Config
	iam          *iam.Iam
	ConfigCenter *configures.ConfCenter
}

func NewService(ctx context.Context) *Service {
	return &Service{
		ctx: ctx,
	}
}

func (s *Service) SetDB(db dal.RDB) {
	s.db = db
}

func (s *Service) SetWatchDB(watchDB dal.RDB) {
	s.watchDB = watchDB
}

func (s *Service) SetCache(cache redis.Client) {
	s.cache = cache
}

func (s *Service) SetIam(iam *iam.Iam) {
	s.iam = iam
}

func (s *Service) WebService() *restful.Container {
	container := restful.NewContainer()

	api := new(restful.WebService)
	getErrFunc := func() errors.CCErrorIf {
		return s.CCErr
	}
	api.Path("/migrate/v3")
	api.Filter(s.Engine.Metric().RestfulMiddleWare)
	api.Filter(rdapi.AllGlobalFilter(getErrFunc))
	api.Produces(restful.MIME_JSON)

	api.Route(api.POST("/authcenter/init").To(s.InitAuthCenter))
	api.Route(api.POST("/migrate/{distribution}/{ownerID}").To(s.migrate))
	api.Route(api.POST("/migrate/system/hostcrossbiz/{ownerID}").To(s.SetSystemConfiguration))
	api.Route(api.POST("/migrate/system/user_config/{key}/{can}").To(s.UserConfigSwitch))
	api.Route(api.GET("/find/system/config_admin").To(s.SearchConfigAdmin))
	api.Route(api.PUT("/update/system/config_admin").To(s.UpdateConfigAdmin))
	api.Route(api.POST("/migrate/specify/version/{distribution}/{ownerID}").To(s.migrateSpecifyVersion))
	api.Route(api.POST("/migrate/config/refresh").To(s.refreshConfig))
	api.Route(api.POST("/migrate/dataid").To(s.migrateDataID))

	container.Add(api)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(s.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

	return container
}

func (s *Service) Healthz(req *restful.Request, resp *restful.Response) {
	meta := metric.HealthMeta{IsHealthy: true}

	// zk health status
	zkItem := metric.HealthItem{IsHealthy: true, Name: types.CCFunctionalityServicediscover}
	if err := s.Engine.Ping(); err != nil {
		zkItem.IsHealthy = false
		zkItem.Message = err.Error()
	}
	meta.Items = append(meta.Items, zkItem)

	// mongodb
	healthItem := metric.NewHealthItem(types.CCFunctionalityMongo, s.db.Ping())
	meta.Items = append(meta.Items, healthItem)

	// redis
	redisItem := metric.NewHealthItem(types.CCFunctionalityRedis, s.cache.Ping(context.Background()).Err())
	meta.Items = append(meta.Items, redisItem)

	for _, item := range meta.Items {
		if item.IsHealthy == false {
			meta.IsHealthy = false
			meta.Message = "admin server is unhealthy"
			break
		}
	}

	info := metric.HealthInfo{
		Module:     types.CC_MODULE_MIGRATE,
		HealthMeta: meta,
		AtTime:     metadata.Now(),
	}

	answer := metric.HealthResponse{
		Code:    common.CCSuccess,
		Data:    info,
		OK:      meta.IsHealthy,
		Result:  meta.IsHealthy,
		Message: meta.Message,
	}
	answer.SetCommonResponse()
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteEntity(answer)
}

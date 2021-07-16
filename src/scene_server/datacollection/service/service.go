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
	"fmt"

	"configcenter/src/common"
	"configcenter/src/common/backbone"
	"configcenter/src/common/errors"
	"configcenter/src/common/metadata"
	"configcenter/src/common/metric"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/types"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/scene_server/datacollection/logics"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/redis"
	"configcenter/src/thirdparty/esbserver"

	"github.com/emicklei/go-restful"
)

// Service impls main logics as service for datacolection app.
type Service struct {
	ctx    context.Context
	engine *backbone.Engine

	db      dal.RDB
	cache   redis.Client
	snapCli redis.Client
	disCli  redis.Client
	netCli  redis.Client

	logics *logics.Logics
}

// NewService creates a new Service object.
func NewService(ctx context.Context, engine *backbone.Engine) *Service {
	return &Service{ctx: ctx, engine: engine}
}

// SetLogics setups logics comm.
func (s *Service) SetLogics(db dal.RDB, esb esbserver.EsbClientInterface) {
	s.logics = logics.NewLogics(s.ctx, s.engine, db, esb)
}

// SetDB setups database.
func (s *Service) SetDB(db dal.RDB) {
	s.db = db
}

// SetCache setups cc main redis.
func (s *Service) SetCache(db redis.Client) {
	s.cache = db
}

// SetSnapCli setups snap redis.
func (s *Service) SetSnapCli(db redis.Client) {
	s.snapCli = db
}

// SetDiscoverCli setups discover redis.
func (s *Service) SetDiscoverCli(db redis.Client) {
	s.disCli = db
}

// SetNetCollectCli setups netcollect redis.
func (s *Service) SetNetCollectCli(db redis.Client) {
	s.netCli = db
}

// WebService setups a new restful web service.
func (s *Service) WebService() *restful.Container {
	container := restful.NewContainer()

	api := new(restful.WebService)
	getErrFunc := func() errors.CCErrorIf {
		return s.engine.CCErr
	}

	api.Path("/collector/v3").Filter(s.engine.Metric().RestfulMiddleWare).Filter(rdapi.AllGlobalFilter(getErrFunc)).Produces(restful.MIME_JSON)

	api.Route(api.POST("/netcollect/device/action/create").To(s.CreateDevice))
	api.Route(api.POST("/netcollect/device/{device_id}/action/update").To(s.UpdateDevice))
	api.Route(api.POST("/netcollect/device/action/batch").To(s.BatchCreateDevice))
	api.Route(api.POST("/netcollect/device/action/search").To(s.SearchDevice))
	api.Route(api.DELETE("/netcollect/device/action/delete").To(s.DeleteDevice))

	api.Route(api.POST("/netcollect/property/action/create").To(s.CreateProperty))
	api.Route(api.POST("/netcollect/property/{netcollect_property_id}/action/update").To(s.UpdateProperty))
	api.Route(api.POST("/netcollect/property/action/batch").To(s.BatchCreateProperty))
	api.Route(api.POST("/netcollect/property/action/search").To(s.SearchProperty))
	api.Route(api.DELETE("/netcollect/property/action/delete").To(s.DeleteProperty))

	api.Route(api.POST("/netcollect/summary/action/search").To(s.SearchReportSummary))
	api.Route(api.POST("/netcollect/report/action/search").To(s.SearchReport))
	api.Route(api.POST("/netcollect/report/action/confirm").To(s.ConfirmReport))
	api.Route(api.POST("/netcollect/history/action/search").To(s.SearchHistory))

	api.Route(api.POST("/netcollect/collector/action/search").To(s.SearchCollector))
	api.Route(api.POST("/netcollect/collector/action/update").To(s.UpdateCollector))
	api.Route(api.POST("/netcollect/collector/action/discover").To(s.DiscoverNetDevice))

	container.Add(api)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(s.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

	return container
}

// Healthz is a HTTP restful interface for health check.
func (s *Service) Healthz(req *restful.Request, resp *restful.Response) {
	// metadata.
	meta := metric.HealthMeta{IsHealthy: true}

	// zookeeper health status info.
	zkItem := metric.HealthItem{
		IsHealthy: true,
		Name:      types.CCFunctionalityServicediscover,
	}
	if err := s.engine.Ping(); err != nil {
		zkItem.IsHealthy = false
		zkItem.Message = err.Error()
	}
	meta.Items = append(meta.Items, zkItem)

	// topo server health status info.
	topoItem := metric.HealthItem{
		IsHealthy: true,
		Name:      types.CC_MODULE_TOPO,
	}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_TOPO); err != nil {
		topoItem.IsHealthy = false
		topoItem.Message = err.Error()
	}
	meta.Items = append(meta.Items, topoItem)

	// mongodb health status info.
	meta.Items = append(meta.Items, metric.NewHealthItem(types.CCFunctionalityMongo, s.db.Ping()))

	// cc main redis health status info.
	meta.Items = append(meta.Items, metric.NewHealthItem(types.CCFunctionalityRedis, s.cache.Ping(context.Background()).Err()))

	// snap redis health status info.
	if s.snapCli != nil {
		meta.Items = append(meta.Items, metric.NewHealthItem(fmt.Sprintf("%s - snapCli", types.CCFunctionalityRedis), s.snapCli.Ping(context.Background()).Err()))
	}

	// discover redis health status info.
	if s.disCli != nil {
		meta.Items = append(meta.Items, metric.NewHealthItem(fmt.Sprintf("%s - disCli", types.CCFunctionalityRedis), s.disCli.Ping(context.Background()).Err()))
	}

	// netcollect redis health status info.
	if s.netCli != nil {
		meta.Items = append(meta.Items, metric.NewHealthItem(fmt.Sprintf("%s - netCli", types.CCFunctionalityRedis), s.netCli.Ping(context.Background()).Err()))
	}

	for _, item := range meta.Items {
		if item.IsHealthy == false {
			meta.IsHealthy = false
			meta.Message = "datacolection server is unhealthy"
			break
		}
	}

	info := metric.HealthInfo{
		Module:     types.CC_MODULE_DATACOLLECTION,
		HealthMeta: meta,
		AtTime:     metadata.Now(),
	}

	healthResp := metric.HealthResponse{
		Code:    common.CCSuccess,
		Data:    info,
		OK:      meta.IsHealthy,
		Result:  meta.IsHealthy,
		Message: meta.Message,
	}
	healthResp.SetCommonResponse()

	resp.Header().Set("Content-Type", "application/json")
	resp.WriteEntity(healthResp)
}

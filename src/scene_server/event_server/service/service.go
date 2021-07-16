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
	"net/http"

	"configcenter/src/ac"
	"configcenter/src/common"
	"configcenter/src/common/backbone"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/common/metric"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/types"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/scene_server/event_server/distribution"
	"configcenter/src/storage/dal"
	"configcenter/src/storage/dal/redis"

	"github.com/emicklei/go-restful"
)

// Service impls main logics as service for datacolection app.
type Service struct {
	ctx    context.Context
	engine *backbone.Engine

	// db is cc main database.
	db dal.RDB

	// cache is cc redis client.
	cache redis.Client

	// distributer is event subscription distributer.
	distributer *distribution.Distributor

	authorizer ac.AuthorizeInterface
}

// NewService creates a new Service object.
func NewService(ctx context.Context, engine *backbone.Engine) *Service {
	return &Service{ctx: ctx, engine: engine}
}

// SetDB setups database.
func (s *Service) SetDB(db dal.RDB) {
	s.db = db
}

// SetCache setups cc main redis.
func (s *Service) SetCache(db redis.Client) {
	s.cache = db
}

func (s *Service) SetAuthorizer(authorizer ac.AuthorizeInterface) {
	s.authorizer = authorizer
}

// SetDistributer setups event subscription distributer.
func (s *Service) SetDistributer(distributer *distribution.Distributor) {
	s.distributer = distributer
}

// WebService setups a new restful web service.
func (s *Service) WebService() *restful.Container {
	container := restful.NewContainer()

	api := new(restful.WebService)
	getErrFunc := func() errors.CCErrorIf {
		return s.engine.CCErr
	}

	api.Path("/event/v3")
	api.Filter(s.engine.Metric().RestfulMiddleWare)
	api.Filter(rdapi.AllGlobalFilter(getErrFunc))
	api.Produces(restful.MIME_JSON)

	s.initService(api)

	container.Add(api)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(s.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

	return container
}

func (s *Service) initService(web *restful.WebService) {

	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  s.engine.CCErr,
		Language: s.engine.Language,
	})

	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/subscribe/search/{ownerID}/{appID}", Handler: s.ListSubscriptions})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/subscribe/{ownerID}/{appID}", Handler: s.Subscribe})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/subscribe/{ownerID}/{appID}/{subscribeID}", Handler: s.UnSubscribe})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/subscribe/{ownerID}/{appID}/{subscribeID}", Handler: s.UpdateSubscription})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/subscribe/ping", Handler: s.Ping})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/subscribe/telnet", Handler: s.Telnet})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/watch/resource/{resource}", Handler: s.WatchEvent})

	utility.AddToRestfulWebService(web)

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

	// mongodb health status info.
	meta.Items = append(meta.Items, metric.NewHealthItem(types.CCFunctionalityMongo, s.db.Ping()))

	// cc main redis health status info.
	meta.Items = append(meta.Items, metric.NewHealthItem(types.CCFunctionalityRedis, s.cache.Ping(context.Background()).Err()))

	for _, item := range meta.Items {
		if item.IsHealthy == false {
			meta.IsHealthy = false
			meta.Message = "event server is unhealthy"
			break
		}
	}

	info := metric.HealthInfo{
		Module:     types.CC_MODULE_EVENTSERVER,
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

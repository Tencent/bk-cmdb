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
	"configcenter/src/ac"
	"configcenter/src/ac/iam"
	"configcenter/src/apimachinery"
	"configcenter/src/apimachinery/discovery"
	"configcenter/src/common/auth"
	"configcenter/src/common/backbone"
	"configcenter/src/common/errors"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/storage/dal/redis"

	"github.com/emicklei/go-restful"
)

// Service service methods
type Service interface {
	WebServices() []*restful.WebService
	SetConfig(engine *backbone.Engine, httpClient HTTPClient, discovery discovery.DiscoveryInterface,
		clientSet apimachinery.ClientSetInterface, cache redis.Client, limiter *Limiter)
}

// NewService create a new service instance
func NewService() Service {
	return new(service)
}

type service struct {
	engine     *backbone.Engine
	client     HTTPClient
	discovery  discovery.DiscoveryInterface
	clientSet  apimachinery.ClientSetInterface
	authorizer ac.AuthorizeInterface
	cache      redis.Client
	limiter    *Limiter
}

func (s *service) SetConfig(engine *backbone.Engine, httpClient HTTPClient, discovery discovery.DiscoveryInterface,
	clientSet apimachinery.ClientSetInterface, cache redis.Client, limiter *Limiter) {
	s.engine = engine
	s.client = httpClient
	s.discovery = discovery
	s.clientSet = clientSet
	s.cache = cache
	s.limiter = limiter
	s.authorizer = iam.NewAuthorizer(clientSet)
}

func (s *service) WebServices() []*restful.WebService {
	getErrFun := func() errors.CCErrorIf {
		return s.engine.CCErr
	}

	ws := &restful.WebService{}
	ws.Path(rootPath)
	ws.Filter(s.engine.Metric().RestfulMiddleWare)
	ws.Filter(rdapi.AllGlobalFilter(getErrFun))
	ws.Filter(rdapi.RequestLogFilter())
	ws.Filter(s.LimiterFilter())
	ws.Produces(restful.MIME_JSON)
	if auth.EnableAuthorize() {
		ws.Filter(s.authFilter(getErrFun))
	}
	ws.Route(ws.POST("/auth/verify").To(s.AuthVerify))
	ws.Route(ws.GET("/auth/business_list").To(s.GetAnyAuthorizedAppList))
	ws.Route(ws.POST("/auth/skip_url").To(s.GetUserNoAuthSkipURL))
	ws.Route(ws.GET("{.*}").Filter(s.URLFilterChan).To(s.Get))
	ws.Route(ws.POST("{.*}").Filter(s.URLFilterChan).To(s.Post))
	ws.Route(ws.PUT("{.*}").Filter(s.URLFilterChan).To(s.Put))
	ws.Route(ws.DELETE("{.*}").Filter(s.URLFilterChan).To(s.Delete))

	allWebServices := make([]*restful.WebService, 0)
	allWebServices = append(allWebServices, ws)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(s.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	allWebServices = append(allWebServices, commonAPI)

	return allWebServices
}

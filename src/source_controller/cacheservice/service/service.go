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
	"net/http"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/backbone"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/language"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/util"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/source_controller/cacheservice/app/options"
	"configcenter/src/source_controller/cacheservice/cache"
	cacheop "configcenter/src/source_controller/cacheservice/cache"
	"configcenter/src/source_controller/cacheservice/event/flow"
	"configcenter/src/source_controller/coreservice/core"
	"configcenter/src/storage/dal/mongo/local"
	"configcenter/src/storage/reflector"
	"configcenter/src/storage/stream"

	"github.com/emicklei/go-restful"
)

// CacheServiceInterface the cache service methods used to init
type CacheServiceInterface interface {
	WebService() *restful.Container
	SetConfig(cfg options.Config, engine *backbone.Engine, err errors.CCErrorIf, language language.CCLanguageIf) error
}

// New create cache service instance
func New() CacheServiceInterface {
	return &cacheService{}
}

// cacheService cache service
type cacheService struct {
	engine      *backbone.Engine
	langFactory map[common.LanguageType]language.DefaultCCLanguageIf
	language    language.CCLanguageIf
	err         errors.CCErrorIf
	cfg         options.Config
	core        core.Core
	cacheSet    *cache.ClientSet
}

func (s *cacheService) SetConfig(cfg options.Config, engine *backbone.Engine, err errors.CCErrorIf, lang language.CCLanguageIf) error {

	s.cfg = cfg
	s.engine = engine

	if nil != err {
		s.err = err
	}

	if nil != lang {
		s.langFactory = make(map[common.LanguageType]language.DefaultCCLanguageIf)
		s.langFactory[common.Chinese] = lang.CreateDefaultCCLanguageIf(string(common.Chinese))
		s.langFactory[common.English] = lang.CreateDefaultCCLanguageIf(string(common.English))
	}

	loopW, loopErr := stream.NewLoopStream(s.cfg.Mongo.GetMongoConf(), engine.ServiceManageInterface)
	if loopErr != nil {
		blog.Errorf("new loop stream failed, err: %v", loopErr)
		return loopErr
	}

	event, eventErr := reflector.NewReflector(s.cfg.Mongo.GetMongoConf())
	if eventErr != nil {
		blog.Errorf("new reflector failed, err: %v", eventErr)
		return eventErr
	}

	watchDB, dbErr := local.NewMgo(s.cfg.WatchMongo.GetMongoConf(), time.Minute)
	if dbErr != nil {
		blog.Errorf("new watch mongo client failed, err: %v", dbErr)
		return dbErr
	}

	c, cacheErr := cacheop.NewCache(event, loopW, engine.ServiceManageInterface, watchDB)
	if cacheErr != nil {
		blog.Errorf("new cache instance failed, err: %v", cacheErr)
		return cacheErr
	}
	s.cacheSet = c

	watcher, watchErr := stream.NewLoopStream(s.cfg.Mongo.GetMongoConf(), engine.ServiceManageInterface)
	if watchErr != nil {
		blog.Errorf("new loop watch stream failed, err: %v", watchErr)
		return watchErr
	}

	ccDB, dbErr := local.NewMgo(s.cfg.Mongo.GetMongoConf(), time.Minute)
	if dbErr != nil {
		blog.Errorf("new cc mongo client failed, err: %v", dbErr)
		return dbErr
	}

	flowErr := flow.NewEvent(watcher, engine.ServiceManageInterface, watchDB, ccDB)
	if flowErr != nil {
		blog.Errorf("new watch event failed, err: %v", flowErr)
		return flowErr
	}

	return nil
}

// WebService the web service
func (s *cacheService) WebService() *restful.Container {

	container := restful.NewContainer()

	getErrFunc := func() errors.CCErrorIf { return s.err }

	api := new(restful.WebService)
	api.Path("/cache/v3").Filter(s.engine.Metric().RestfulMiddleWare).Filter(rdapi.AllGlobalFilter(getErrFunc)).
		Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON)

	// init service actions
	s.initService(api)
	container.Add(api)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(s.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

	return container
}

func (s *cacheService) Language(header http.Header) language.DefaultCCLanguageIf {
	lang := util.GetLanguage(header)
	l, exist := s.langFactory[common.LanguageType(lang)]
	if !exist {
		return s.langFactory[common.Chinese]
	}
	return l
}

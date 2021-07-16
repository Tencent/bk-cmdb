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

	"configcenter/src/common"
	"configcenter/src/common/backbone"
	"configcenter/src/common/errors"
	"configcenter/src/common/language"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/util"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/source_controller/coreservice/app/options"
	"configcenter/src/source_controller/coreservice/core"
	"configcenter/src/source_controller/coreservice/core/association"
	"configcenter/src/source_controller/coreservice/core/auditlog"
	"configcenter/src/source_controller/coreservice/core/auth"
	"configcenter/src/source_controller/coreservice/core/cloud"
	coreCommon "configcenter/src/source_controller/coreservice/core/common"
	"configcenter/src/source_controller/coreservice/core/datasynchronize"
	e "configcenter/src/source_controller/coreservice/core/event"
	"configcenter/src/source_controller/coreservice/core/host"
	"configcenter/src/source_controller/coreservice/core/hostapplyrule"
	"configcenter/src/source_controller/coreservice/core/instances"
	"configcenter/src/source_controller/coreservice/core/label"
	"configcenter/src/source_controller/coreservice/core/mainline"
	"configcenter/src/source_controller/coreservice/core/model"
	"configcenter/src/source_controller/coreservice/core/operation"
	"configcenter/src/source_controller/coreservice/core/process"
	"configcenter/src/source_controller/coreservice/core/settemplate"
	dbSystem "configcenter/src/source_controller/coreservice/core/system"
	"configcenter/src/storage/driver/mongodb"
	"configcenter/src/storage/driver/redis"

	"github.com/emicklei/go-restful"
)

// CoreServiceInterface the topo service methods used to init
type CoreServiceInterface interface {
	WebService() *restful.Container
	SetConfig(cfg options.Config, engine *backbone.Engine, err errors.CCErrorIf, language language.CCLanguageIf) error
}

// New create topo service instance
func New() CoreServiceInterface {
	return &coreService{}
}

// coreService topo service
type coreService struct {
	engine      *backbone.Engine
	langFactory map[common.LanguageType]language.DefaultCCLanguageIf
	language    language.CCLanguageIf
	err         errors.CCErrorIf
	cfg         options.Config
	core        core.Core
}

func (s *coreService) SetConfig(cfg options.Config, engine *backbone.Engine, err errors.CCErrorIf, lang language.CCLanguageIf) error {

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

	/* db, dbErr := local.NewMgo(s.cfg.Mongo.GetMongoConf(), time.Minute)
	if dbErr != nil {
		blog.Errorf("failed to connect the txc server, error info is %s", dbErr.Error())
		return dbErr
	}

	 cache, cacheRrr := dalredis.NewFromConfig(cfg.Redis)
	if cacheRrr != nil {
		blog.Errorf("new redis client failed, err: %v", cacheRrr)
		return cacheRrr
	}
	initErr := db.InitTxnManager(cache)
	if initErr != nil {
		blog.Errorf("failed to init txn manager, error info is %v", initErr)
		return initErr
	}
	mongodb.Client() = db
	s.rds = cache */

	// connect the remote mongodb
	instance := instances.New(s, lang, engine.CoreAPI)
	hostApplyRuleCore := hostapplyrule.New(instance)
	s.core = core.New(
		model.New(s, lang),
		instance,
		association.New(s),
		datasynchronize.New(s),
		mainline.New(lang),
		host.New(s, hostApplyRuleCore),
		auditlog.New(),
		process.New(s),
		label.New(),
		settemplate.New(),
		operation.New(),
		hostApplyRuleCore,
		dbSystem.New(),
		cloud.New(mongodb.Client()),
		auth.New(mongodb.Client()),
		e.New(mongodb.Client(), redis.Client()),
		coreCommon.New(),
	)
	return nil
}

// WebService the web service
func (s *coreService) WebService() *restful.Container {

	container := restful.NewContainer()
	getErrFunc := func() errors.CCErrorIf {
		return s.err
	}
	api := new(restful.WebService)
	api.Path("/api/v3").Filter(s.engine.Metric().RestfulMiddleWare).Filter(rdapi.AllGlobalFilter(getErrFunc)).Produces(restful.MIME_JSON).Consumes(restful.MIME_JSON)
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

func (s *coreService) Language(header http.Header) language.DefaultCCLanguageIf {
	lang := util.GetLanguage(header)
	l, exist := s.langFactory[common.LanguageType(lang)]
	if !exist {
		return s.langFactory[common.Chinese]
	}
	return l
}

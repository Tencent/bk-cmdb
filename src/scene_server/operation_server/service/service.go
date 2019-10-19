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

	"configcenter/src/auth/authcenter"
	"configcenter/src/auth/extensions"
	"configcenter/src/common"
	"configcenter/src/common/backbone"
	cc "configcenter/src/common/backbone/configcenter"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/language"
	"configcenter/src/common/metadata"
	"configcenter/src/common/metric"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/types"
	"configcenter/src/common/util"
	"configcenter/src/scene_server/operation_server/app/options"
	"configcenter/src/scene_server/operation_server/logics"
	"configcenter/src/storage/dal/mongo"

	"github.com/emicklei/go-restful"
)

type srvComm struct {
	header        http.Header
	rid           string
	ccErr         errors.DefaultCCErrorIf
	ccLang        language.DefaultCCLanguageIf
	ctx           context.Context
	ctxCancelFunc context.CancelFunc
	user          string
	ownerID       string
	lgc           *logics.Logics
}

type OperationServer struct {
	*backbone.Engine
	Config      *options.Config
	ConfigMap   map[string]string
	AuthManager *extensions.AuthManager
	Logic       *logics.Logic
}

func (o *OperationServer) newSrvComm(header http.Header) *srvComm {
	rid := util.GetHTTPCCRequestID(header)
	lang := util.GetLanguage(header)
	ctx, cancel := o.Engine.CCCtx.WithCancel()
	ctx = context.WithValue(ctx, common.ContextRequestIDField, rid)

	return &srvComm{
		header:        header,
		rid:           util.GetHTTPCCRequestID(header),
		ccErr:         o.CCErr.CreateDefaultCCErrorIf(lang),
		ccLang:        o.Language.CreateDefaultCCLanguageIf(lang),
		ctx:           ctx,
		ctxCancelFunc: cancel,
		user:          util.GetUser(header),
		ownerID:       util.GetOwnerID(header),
		lgc:           logics.NewLogics(o.Engine, header, o.AuthManager),
	}
}

func (o *OperationServer) WebService() *restful.Container {

	getErrFunc := func() errors.CCErrorIf {
		return o.Engine.CCErr
	}

	api := new(restful.WebService)
	api.Path("/operation/v3").Filter(rdapi.AllGlobalFilter(getErrFunc)).Produces(restful.MIME_JSON)
	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)

	o.newOperationService(api)
	container := restful.NewContainer()
	container.Add(api)

	healthzAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	healthzAPI.Route(healthzAPI.GET("/healthz").To(o.Healthz))
	container.Add(healthzAPI)

	return container
}

func (o *OperationServer) newOperationService(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  o.Engine.CCErr,
		Language: o.Engine.Language,
	})

	// service category
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/operation/chart", Handler: o.CreateOperationChart})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/operation/chart/{id}", Handler: o.DeleteOperationChart})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/update/operation/chart", Handler: o.UpdateOperationChart})
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/search/operation/chart", Handler: o.SearchOperationChart})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/search/operation/chart/data", Handler: o.SearchChartData})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/update/operation/chart/position", Handler: o.UpdateChartPosition})

	utility.AddToRestfulWebService(web)
}

func (o *OperationServer) Healthz(req *restful.Request, resp *restful.Response) {
	meta := metric.HealthMeta{IsHealthy: true}

	// zk health status
	zkItem := metric.HealthItem{IsHealthy: true, Name: types.CCFunctionalityServicediscover}
	if err := o.Engine.Ping(); err != nil {
		zkItem.IsHealthy = false
		zkItem.Message = err.Error()
	}
	meta.Items = append(meta.Items, zkItem)

	// coreservice
	coreSrv := metric.HealthItem{IsHealthy: true, Name: types.CCModuleCoerService.Name}
	if _, err := o.Engine.CoreAPI.Healthz().HealthCheck(coreSrv.Name); err != nil {
		coreSrv.IsHealthy = false
		coreSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, coreSrv)

	for _, item := range meta.Items {
		if item.IsHealthy == false {
			meta.IsHealthy = false
			meta.Message = "operation server is unhealthy"
			break
		}
	}

	info := metric.HealthInfo{
		Module:     types.CCModuleOperation.Name,
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
	resp.Header().Set("Content-Type", "application/json")
	resp.WriteEntity(answer)
}

func (o *OperationServer) OnOperationConfigUpdate(previous, current cc.ProcessConfig) {
	var err error

	cfg := mongo.ParseConfigFromKV("mongodb", current.ConfigMap)
	o.Config = &options.Config{
		Mongo: cfg,
	}
	o.Config.ConfigMap = current.ConfigMap

	o.Config.Auth, err = authcenter.ParseConfigFromKV("auth", current.ConfigMap)
	if err != nil {
		blog.Warnf("parse auth center config failed: %v", err)
	}
}

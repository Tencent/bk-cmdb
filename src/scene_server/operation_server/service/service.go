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
	goErr "errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"configcenter/src/ac/extensions"
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
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/scene_server/operation_server/app/options"
	"configcenter/src/scene_server/operation_server/logics"
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
		lgc:           logics.NewLogics(o.Engine, header, o.AuthManager, o.Config.Timer),
	}
}

func (o *OperationServer) WebService() *restful.Container {

	getErrFunc := func() errors.CCErrorIf {
		return o.Engine.CCErr
	}

	api := new(restful.WebService)
	api.Path("/operation/v3").Filter(o.Engine.Metric().RestfulMiddleWare).Filter(rdapi.AllGlobalFilter(getErrFunc)).Produces(restful.MIME_JSON)
	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)

	o.newOperationService(api)
	container := restful.NewContainer()
	container.Add(api)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(o.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

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
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/findmany/operation/chart", Handler: o.SearchOperationChart})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/operation/chart/data", Handler: o.SearchChartData})
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
	coreSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_CORESERVICE}
	if _, err := o.Engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_CORESERVICE); err != nil {
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
		Module:     types.CC_MODULE_OPERATION,
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

func (o *OperationServer) OnOperationConfigUpdate(previous, current cc.ProcessConfig) {
	var err error
	o.Config = &options.Config{}
	o.Config.Timer, err = o.ParseTimerConfigFromKV("operationServer.timer", nil)
	if err != nil {
		blog.Errorf("parse timer config failed, err: %v", err)
		return
	}
}

func (o *OperationServer) ParseTimerConfigFromKV(prefix string, configMap map[string]string) (string, error) {
	// 若是timer没配置，或者解析失败，给一个默认的定时时间
	defaultSpec := "30 0 * * *"
	specStr, err := cc.String(prefix + ".spec")
	if err != nil {
		blog.Errorf("parse timer config failed, missing 'spec' configuration for timer, set timer-spec default value: 00:30")
		return defaultSpec, nil
	}
	spec, err := parseTimerConfig(specStr)
	if err != nil || spec == "" {
		blog.Errorf("parse timer config failed, set timer-spec default value: 00:30, err: %v", err)
		return defaultSpec, nil
	}

	return spec, nil
}

func parseTimerConfig(spec string) (string, error) {
	matched, err := regexp.MatchString(common.TimerPattern, spec)
	if err != nil || !matched {
		blog.Errorf("parse timer config failed, 'spec' not match required rules, err: %v", err)
		return "", goErr.New("'spec' not match required rules")
	}

	numArray := strings.Split(spec, ":")
	hour := numArray[0]
	intHour, err := strconv.Atoi(hour)
	if err != nil {
		blog.Errorf("parse timer config failed, got invalid hour data, err: %v", err)
		return "", goErr.New("parse time config failed, got invalid hour data")
	}
	if intHour < 0 || intHour > 23 {
		blog.Errorf("parse timer config failed, got invalid hour data, err: %v", err)
		return "", goErr.New("'parse time config failed, got invalid hour data, should between 0-23")
	}
	minute := numArray[1]
	intMinute, err := strconv.Atoi(minute)
	if err != nil {
		blog.Errorf("parse timer config failed, got invalid minute data, err: %v", err)
		return "", goErr.New("parse time config failed, got invalid minute data")
	}
	if intMinute < 0 || intMinute > 59 {
		blog.Errorf("parse timer config failed, got invalid minute data, err: %v", err)
		return "", goErr.New("parse time config failed, got invalid minute data, should between 0-59")
	}

	spec = fmt.Sprintf("%d %d * * *", intMinute, intHour)
	return spec, nil
}

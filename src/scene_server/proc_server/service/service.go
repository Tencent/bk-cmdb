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

	"configcenter/src/ac/extensions"
	"configcenter/src/common"
	"configcenter/src/common/backbone"
	cfnc "configcenter/src/common/backbone/configcenter"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/common/metric"
	"configcenter/src/common/rdapi"
	"configcenter/src/common/types"
	"configcenter/src/common/util"
	"configcenter/src/common/webservice/restfulservice"
	"configcenter/src/scene_server/proc_server/app/options"
	"configcenter/src/scene_server/proc_server/logics"
	"configcenter/src/thirdparty/esbserver"
	"configcenter/src/thirdparty/esbserver/esbutil"

	"github.com/emicklei/go-restful"
)

type ProcServer struct {
	*backbone.Engine
	EsbConfigChn       chan esbutil.EsbConfig
	Config             *options.Config
	EsbSrv             esbserver.EsbClientInterface
	procHostInstConfig logics.ProcHostInstConfig
	ConfigMap          map[string]string
	AuthManager        *extensions.AuthManager
	Logic              *logics.Logic
}

func (ps *ProcServer) WebService() *restful.Container {
	getErrFunc := func() errors.CCErrorIf {
		return ps.Engine.CCErr
	}

	api := new(restful.WebService)
	api.Path("/process/v3")
	api.Filter(ps.Engine.Metric().RestfulMiddleWare)
	api.Filter(rdapi.AllGlobalFilter(getErrFunc))
	api.Produces(restful.MIME_JSON)
	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)

	ps.newProcessService(api)
	container := restful.NewContainer()
	container.Add(api)

	// common api
	commonAPI := new(restful.WebService).Produces(restful.MIME_JSON)
	commonAPI.Route(commonAPI.GET("/healthz").To(ps.Healthz))
	commonAPI.Route(commonAPI.GET("/version").To(restfulservice.Version))
	container.Add(commonAPI)

	return container
}

func (ps *ProcServer) newProcessService(web *restful.WebService) {
	utility := rest.NewRestUtility(rest.Config{
		ErrorIf:  ps.Engine.CCErr,
		Language: ps.Engine.Language,
	})

	// service category
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_category", Handler: ps.ListServiceCategory})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_category/with_statistics", Handler: ps.ListServiceCategoryWithStatistics})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/proc/service_category", Handler: ps.CreateServiceCategory})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/proc/service_category", Handler: ps.UpdateServiceCategory})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/proc/service_category", Handler: ps.DeleteServiceCategory})

	// service template
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/proc/service_template", Handler: ps.CreateServiceTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/proc/service_template", Handler: ps.UpdateServiceTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/find/proc/service_template/{service_template_id}", Handler: ps.GetServiceTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodGet, Path: "/find/proc/service_template/{service_template_id}/detail", Handler: ps.GetServiceTemplateDetail})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_template", Handler: ps.ListServiceTemplates})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/proc/service_template", Handler: ps.DeleteServiceTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_template/count_info/biz/{bk_biz_id}", Handler: ps.FindServiceTemplateCountInfo})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_template/sync_status/biz/{bk_biz_id}", Handler: ps.GetServiceTemplateSyncStatus})

	// process template
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/createmany/proc/proc_template", Handler: ps.CreateProcessTemplateBatch})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/proc/proc_template", Handler: ps.UpdateProcessTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/deletemany/proc/proc_template", Handler: ps.DeleteProcessTemplateBatch})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/proc/proc_template/id/{processTemplateID}", Handler: ps.GetProcessTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/proc_template", Handler: ps.ListProcessTemplate})

	// service instance
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/proc/service_instance", Handler: ps.CreateServiceInstances})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/proc/service_instance/preview", Handler: ps.CreateServiceInstancesPreview})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_instance", Handler: ps.SearchServiceInstancesInModule})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/web/service_instance", Handler: ps.SearchServiceInstancesInModuleWeb})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service/set_template/list_service_instance/biz/{bk_biz_id}", Handler: ps.SearchServiceInstancesBySetTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_instance/with_host", Handler: ps.ListServiceInstancesWithHost})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/web/service_instance/with_host", Handler: ps.ListServiceInstancesWithHostWeb})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_instance/details", Handler: ps.ListServiceInstancesDetails})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/updatemany/proc/service_instance/biz/{bk_biz_id}", Handler: ps.UpdateServiceInstances})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/deletemany/proc/service_instance", Handler: ps.DeleteServiceInstance})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/deletemany/proc/service_instance/preview", Handler: ps.DeleteServiceInstancePreview})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/find/proc/service_instance/difference", Handler: ps.DiffServiceInstanceWithTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/proc/service_instance/sync", Handler: ps.SyncServiceInstanceByTemplate})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/createmany/proc/service_instance/labels", Handler: ps.ServiceInstanceAddLabels})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/deletemany/proc/service_instance/labels", Handler: ps.ServiceInstanceRemoveLabels})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/service_instance/labels/aggregation", Handler: ps.ServiceInstanceLabelsAggregation})

	// process instance
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/create/proc/process_instance", Handler: ps.CreateProcessInstances})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/proc/process_instance", Handler: ps.UpdateProcessInstances})
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/proc/process_instance", Handler: ps.DeleteProcessInstance})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/process_instance", Handler: ps.ListProcessInstances})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/process_related_info/biz/{bk_biz_id}", Handler: ps.ListProcessRelatedInfo})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/process_instance/name_ids", Handler: ps.ListProcessInstancesNameIDsInModule})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/process_instance/detail/by_ids", Handler: ps.ListProcessInstancesDetailsByIDs})
	utility.AddHandler(rest.Action{Verb: http.MethodPost, Path: "/findmany/proc/process_instance/detail/biz/{bk_biz_id}", Handler: ps.ListProcessInstancesDetails})
	utility.AddHandler(rest.Action{Verb: http.MethodPut, Path: "/update/proc/process_instance/by_ids", Handler: ps.UpdateProcessInstancesByIDs})

	// module
	utility.AddHandler(rest.Action{Verb: http.MethodDelete, Path: "/delete/proc/template_binding_on_module", Handler: ps.RemoveTemplateBindingOnModule})

	utility.AddToRestfulWebService(web)
}

func (ps *ProcServer) Healthz(req *restful.Request, resp *restful.Response) {
	meta := metric.HealthMeta{IsHealthy: true}

	// zk health status
	zkItem := metric.HealthItem{IsHealthy: true, Name: types.CCFunctionalityServicediscover}
	if err := ps.Engine.Ping(); err != nil {
		zkItem.IsHealthy = false
		zkItem.Message = err.Error()
	}
	meta.Items = append(meta.Items, zkItem)

	// coreservice
	coreSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_CORESERVICE}
	if _, err := ps.Engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_CORESERVICE); err != nil {
		coreSrv.IsHealthy = false
		coreSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, coreSrv)

	for _, item := range meta.Items {
		if !item.IsHealthy {
			meta.IsHealthy = false
			meta.Message = "proc server is unhealthy"
			break
		}
	}

	info := metric.HealthInfo{
		Module:     types.CC_MODULE_PROC,
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
	_ = resp.WriteEntity(answer)
}

func (ps *ProcServer) OnProcessConfigUpdate(previous, current cfnc.ProcessConfig) {
	esbAddr, addrErr := cfnc.String("procServer.esb.addr")
	esbAppCode, appCodeErr := cfnc.String("procServer.esb.appCode")
	esbAppSecret, appSecretErr := cfnc.String("procServer.esb.appSecret")
	if addrErr == nil && appCodeErr == nil && appSecretErr == nil {
		go func() {
			ps.EsbConfigChn <- esbutil.EsbConfig{Addrs: esbAddr, AppCode: esbAppCode, AppSecret: esbAppSecret}
		}()
	}

	ps.Config = &options.Config{}

	hostInstPrefix := "procServer.host-instance"
	procHostInstConfig := &ps.procHostInstConfig
	maxEventCountVal, err := cfnc.String(hostInstPrefix + ".maxEventCount")
	if err == nil {
		eventCount, err := util.GetIntByInterface(maxEventCountVal)
		if nil == err {
			procHostInstConfig.MaxEventCount = eventCount
		}
	}

	maxModuleIDCountVal, err := cfnc.String(hostInstPrefix + ".maxModuleIDCount")
	if err == nil {
		midCount, err := util.GetIntByInterface(maxModuleIDCountVal)
		if nil == err {
			procHostInstConfig.MaxRefreshModuleCount = midCount
		}
	}

	getModuleIDInterval, err := cfnc.String(hostInstPrefix + ".getModuleIDInterval")
	if err == nil {
		getMidInterval, err := util.GetIntByInterface(getModuleIDInterval)
		if nil == err {
			procHostInstConfig.GetModuleIDInterval = time.Duration(getMidInterval) * time.Second
		}
	}
}

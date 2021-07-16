/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"configcenter/src/common"
	"configcenter/src/common/metadata"
	"configcenter/src/common/metric"
	"configcenter/src/common/types"

	"github.com/emicklei/go-restful"
)

func (s *service) Healthz(req *restful.Request, resp *restful.Response) {
	meta := metric.HealthMeta{IsHealthy: true}

	// zk health status
	zkItem := metric.HealthItem{IsHealthy: true, Name: types.CCFunctionalityServicediscover}
	if err := s.engine.Ping(); err != nil {
		zkItem.IsHealthy = false
		zkItem.Message = err.Error()
	}
	meta.Items = append(meta.Items, zkItem)

	// topo server
	topoSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_TOPO}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_TOPO); err != nil {
		topoSrv.IsHealthy = false
		topoSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, topoSrv)

	// host server
	hostSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_HOST}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_HOST); err != nil {
		hostSrv.IsHealthy = false
		hostSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, hostSrv)

	// proc server
	procSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_PROC}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_PROC); err != nil {
		procSrv.IsHealthy = false
		procSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, procSrv)

	// event server
	eventSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_EVENTSERVER}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_EVENTSERVER); err != nil {
		eventSrv.IsHealthy = false
		eventSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, eventSrv)

	// data collection
	dataCollection := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_DATACOLLECTION}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_DATACOLLECTION); err != nil {
		dataCollection.IsHealthy = false
		dataCollection.Message = err.Error()
	}
	meta.Items = append(meta.Items, dataCollection)

	// operation server
	operationSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_OPERATION}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_OPERATION); err != nil {
		operationSrv.IsHealthy = false
		operationSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, operationSrv)

	// task server
	taskSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_TASK}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_TASK); err != nil {
		taskSrv.IsHealthy = false
		taskSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, taskSrv)

	// cloud server
	cloudSrv := metric.HealthItem{IsHealthy: true, Name: types.CC_MODULE_CLOUD}
	if _, err := s.engine.CoreAPI.Healthz().HealthCheck(types.CC_MODULE_CLOUD); err != nil {
		cloudSrv.IsHealthy = false
		cloudSrv.Message = err.Error()
	}
	meta.Items = append(meta.Items, cloudSrv)

	for _, item := range meta.Items {
		if item.IsHealthy == false {
			meta.IsHealthy = false
			meta.Message = "api server is unhealthy"
			break
		}
	}

	info := metric.HealthInfo{
		Module:     types.CC_MODULE_APISERVER,
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
	resp.WriteJson(answer, "application/json")
}

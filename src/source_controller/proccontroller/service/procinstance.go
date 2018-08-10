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
	"strings"

	"github.com/emicklei/go-restful"
	"github.com/gin-gonic/gin/json"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	meta "configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

func (ps *ProctrlServer) CreateProcInstanceModel(req *restful.Request, resp *restful.Response) {
	language := util.GetLanguage(req.Request.Header)
	defErr := ps.Core.CCErr.CreateDefaultCCErrorIf(language)

	reqParam := make([]meta.ProcInstanceModel, 0)
	if err := json.NewDecoder(req.Request.Body).Decode(&reqParam); err != nil {
		blog.Errorf("create process instance model failed, decode request body err: %v", err)
		resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}

	blog.V(3).Infof("will create process instance model: %+v", reqParam)
	if err := ps.DbInstance.InsertMuti(common.BKTableNameProcInstanceModel, reqParam); err != nil {
		blog.Errorf("create process instance model failed. err: %v", err)
		resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: defErr.Error(common.CCErrProcCreateInstanceModel)})
		return
	}

	resp.WriteEntity(meta.NewSuccessResp(nil))
}

func (ps *ProctrlServer) GetProcInstanceModel(req *restful.Request, resp *restful.Response) {
	language := util.GetLanguage(req.Request.Header)
	defErr := ps.Core.CCErr.CreateDefaultCCErrorIf(language)

	reqParam := new(meta.QueryInput)
	if err := json.NewDecoder(req.Request.Body).Decode(&reqParam); err != nil {
		blog.Errorf("get process instance model failed, decode request body err: %v", err)
		resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}
	cnt, err := ps.DbInstance.GetCntByCondition(common.BKTableNameProcInstanceModel, reqParam.Condition)
	if err != nil {
		blog.Errorf("get process instance model failed. err: %v", err)
		resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: defErr.Error(common.CCErrProcGetInstanceModel)})
		return
	}
	blog.V(3).Infof("will get process instance model. condition: %v", reqParam)
	data := make([]meta.ProcInstanceModel, 0)
	err = ps.DbInstance.GetMutilByCondition(common.BKTableNameProcInstanceModel, strings.Split(reqParam.Fields, ","), reqParam.Condition, &data, reqParam.Sort, reqParam.Start, reqParam.Limit)
	if err != nil {
		blog.Errorf("get process instance model failed. err: %v", err)
		resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: defErr.Error(common.CCErrProcGetInstanceModel)})
		return
	}
	ret := meta.ProcInstModelResult{
		BaseResp: meta.SuccessBaseResp,
	}
	ret.Data.Info = data
	ret.Data.Count = cnt
	resp.WriteEntity(ret)
}

func (ps *ProctrlServer) DeleteProcInstanceModel(req *restful.Request, resp *restful.Response) {
	language := util.GetLanguage(req.Request.Header)
	defErr := ps.Core.CCErr.CreateDefaultCCErrorIf(language)

	reqParam := make(map[string]interface{})
	if err := json.NewDecoder(req.Request.Body).Decode(&reqParam); err != nil {
		blog.Errorf("delete process instance model failed, decode request body err: %v", err)
		resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}

	blog.V(3).Infof("will delete process instance model. condition: %+v", reqParam)
	if err := ps.DbInstance.DelByCondition(common.BKTableNameProcInstanceModel, reqParam); err != nil {
		blog.Errorf("delete process instance model failed. err: %v", err)
		resp.WriteError(http.StatusInternalServerError, &meta.RespError{Msg: defErr.Error(common.CCErrProcDeleteInstanceModel)})
		return
	}

	resp.WriteEntity(meta.NewSuccessResp(nil))
}

func (ps *ProctrlServer) RegisterProcInstaceDetail(req *restful.Request, resp *restful.Response) {
	language := util.GetLanguage(req.Request.Header)
	defErr := ps.Core.CCErr.CreateDefaultCCErrorIf(language)

	input := new(meta.GseProcRequest)
	if err := json.NewDecoder(req.Request.Body).Decode(&input); err != nil {
		blog.Errorf("register  process instance detail failed, decode request body err: %v", err)
		resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}
	for _, gseHost := range input.Hosts {
		conds := common.KvMap{common.BKAppIDField: input.AppID, common.BKProcIDField: input.ProcID, common.BKModuleIDField: input.ModuleID, common.BKHostIDField: gseHost.HostID}
		cnt, err := ps.DbInstance.GetCntByCondition(common.BKTableNameProcInstaceDetail, conds)
		if nil != err {
			blog.Errorf("register  process instance detail get info error: %s", err.Error())
			resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommDBSelectFailed)})
			return
		}
		detail := new(meta.ProcInstanceDetail)
		detail.AppID = input.AppID
		detail.Meta = input.Meta
		detail.ProcID = input.ProcID
		detail.ModuleID = input.ModuleID
		detail.HostID = gseHost.HostID
		detail.Spec = input.Spec
		detail.Hosts = append(detail.Hosts, gseHost)
		detail.Status = meta.ProcInstanceDetailStatusRegisterSucc //1 register gse sucess, 2 register error need retry 3 unregister error need retry
		if 0 == cnt {
			_, err = ps.DbInstance.Insert(common.BKTableNameProcInstaceDetail, detail)
		} else {
			err = ps.DbInstance.UpdateByCondition(common.BKTableNameProcInstaceDetail, detail, conds)
		}
		if nil != err {
			blog.Errorf("register  process instance detail save info error: %s", err.Error())
			resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommDBUpdateFailed)})
			return
		}
	}
	resp.WriteEntity(meta.NewSuccessResp(nil))
}

func (ps *ProctrlServer) ModifyRegisterProcInstanceDetail(req *restful.Request, resp *restful.Response) {
	language := util.GetLanguage(req.Request.Header)
	defErr := ps.Core.CCErr.CreateDefaultCCErrorIf(language)

	input := new(meta.ModifyProcInstanceStatus)
	if err := json.NewDecoder(req.Request.Body).Decode(&input); err != nil {
		blog.Errorf("modify register  process instance detail failed, decode request body err: %v", err)
		resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommJSONUnmarshalFailed)})
		return
	}

	err := ps.DbInstance.UpdateByCondition(common.BKTableNameProcInstaceDetail, input.Data, input.Conds)
	if nil != err {
		blog.Errorf("update register  process instance detail  info error: %s, input:%s", err.Error(), input)
		resp.WriteError(http.StatusBadRequest, &meta.RespError{Msg: defErr.Error(common.CCErrCommDBUpdateFailed)})
		return
	}
	resp.WriteEntity(meta.NewSuccessResp(nil))
}

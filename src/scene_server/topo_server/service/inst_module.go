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
	"fmt"
	"strconv"
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	gparams "configcenter/src/common/paraparse"
	"configcenter/src/scene_server/topo_server/core/types"
)

// CreateModule create a new module
func (s *Service) CreateModule(params types.ContextParams, pathParams, queryParams ParamsGetter, data mapstr.MapStr) (interface{}, error) {

	obj, err := s.Core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDModule)
	if nil != err {
		blog.Errorf("failed to search the set, %s", err.Error())
		return nil, err
	}

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	setID, err := strconv.ParseInt(pathParams("set_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the set id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "set id")
	}

	module, err := s.Core.ModuleOperation().CreateModule(params, obj, bizID, setID, data)
	if err != nil {
		return nil, fmt.Errorf("create module failed, err: %+v", module)
	}

	moduleID, err := module.GetInstID()
	if err != nil {
		return nil, fmt.Errorf("unexpected error, create module success, but get id failed, err: %+v", err)
	}

	// auth: register module to iam
	if err := s.AuthManager.RegisterModuleByID(params.Context, params.Header, moduleID); err != nil {
		return nil, fmt.Errorf("register module failed, err: %+v", err)
	}

	return module, nil
}

// DeleteModule delete the module
func (s *Service) DeleteModule(params types.ContextParams, pathParams, queryParams ParamsGetter, data mapstr.MapStr) (interface{}, error) {

	obj, err := s.Core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDModule)
	if nil != err {
		blog.Errorf("failed to search the module, %s", err.Error())
		return nil, err
	}

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	setID, err := strconv.ParseInt(pathParams("set_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the set id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "set id")
	}

	moduleID, err := strconv.ParseInt(pathParams("module_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the module id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "module id")
	}

	err = s.Core.ModuleOperation().DeleteModule(params, obj, bizID, []int64{setID}, []int64{moduleID})
	if err != nil {
		return nil, fmt.Errorf("delete module failed, err: %+v", err)
	}

	// auth: deregister module to iam
	if err := s.AuthManager.DeregisterModuleByID(params.Context, params.Header, moduleID); err != nil {
		return nil, fmt.Errorf("deregister module failed, err: %+v", err)
	}
	return nil, nil
}

// UpdateModule update the module
func (s *Service) UpdateModule(params types.ContextParams, pathParams, queryParams ParamsGetter, data mapstr.MapStr) (interface{}, error) {

	obj, err := s.Core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDModule)
	if nil != err {
		blog.Errorf("failed to search the module, %s", err.Error())
		return nil, err
	}

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	setID, err := strconv.ParseInt(pathParams("set_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the set id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "set id")
	}

	moduleID, err := strconv.ParseInt(pathParams("module_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the module id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "module id")
	}

	err = s.Core.ModuleOperation().UpdateModule(params, data, obj, bizID, setID, moduleID)
	if err != nil {
		return nil, fmt.Errorf("update module failed, err: %+v", err)
	}

	// auth: update registered module to iam
	if err := s.AuthManager.UpdateRegisteredModuleByID(params.Context, params.Header, moduleID); err != nil {
		return nil, fmt.Errorf("update registered module failed, err: %+v", err)
	}

	return nil, nil
}

// SearchModule search the modules
func (s *Service) SearchModule(params types.ContextParams, pathParams, queryParams ParamsGetter, data mapstr.MapStr) (interface{}, error) {

	obj, err := s.Core.ObjectOperation().FindSingleObject(params, common.BKInnerObjIDModule)
	if nil != err {
		blog.Errorf("failed to search the module, %s", err.Error())
		return nil, err
	}

	bizID, err := strconv.ParseInt(pathParams("app_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the biz id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "business id")
	}

	setID, err := strconv.ParseInt(pathParams("set_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-module]failed to parse the set id, error info is %s", err.Error())
		return nil, params.Err.Errorf(common.CCErrCommParamsNeedInt, "set id")
	}

	paramsCond := &gparams.SearchParams{
		Condition: mapstr.New(),
	}
	if err = data.MarshalJSONInto(paramsCond); nil != err {
		return nil, err
	}

	paramsCond.Condition[common.BKAppIDField] = bizID
	paramsCond.Condition[common.BKSetIDField] = setID

	queryCond := &metadata.QueryInput{}
	queryCond.Condition = paramsCond.Condition
	queryCond.Fields = strings.Join(paramsCond.Fields, ",")
	page := metadata.ParsePage(paramsCond.Page)
	queryCond.Limit = page.Limit
	queryCond.Sort = page.Sort
	queryCond.Start = page.Start

	cnt, instItems, err := s.Core.ModuleOperation().FindModule(params, obj, queryCond)
	if nil != err {
		blog.Errorf("[api-business] failed to find the objects(%s), error info is %s", pathParams("obj_id"), err.Error())
		return nil, err
	}

	result := mapstr.MapStr{}
	result.Set("count", cnt)
	result.Set("info", instItems)

	return result, nil
}

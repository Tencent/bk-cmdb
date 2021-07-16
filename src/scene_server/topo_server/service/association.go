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
	"bytes"
	"io/ioutil"
	"sort"
	"strconv"

	"configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/auth"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/metadata"
	"configcenter/src/scene_server/topo_server/core/model"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// CreateMainLineObject create a new model in the main line topo
func (s *Service) CreateMainLineObject(ctx *rest.Contexts) {
	data := make(map[string]interface{})
	if err := ctx.DecodeInto(&data); err != nil {
		ctx.RespAutoError(err)
		return
	}
	mainLineAssociation := &metadata.Association{}
	_, err := mainLineAssociation.Parse(data)
	if nil != err {
		blog.Errorf("[api-asst] failed to parse the data(%#v), error info is %s, rid: %s", data, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "mainline object"))
		return
	}

	var ret model.Object
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		ret, err = s.Core.AssociationOperation().CreateMainlineAssociation(ctx.Kit, mainLineAssociation, s.Config.BusinessTopoLevelMax)
		if err != nil {
			blog.Errorf("create mainline object: %s failed, err: %v, rid: %s", mainLineAssociation.ObjectID, err, ctx.Kit.Rid)
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(ret)

}

// DeleteMainLineObject delete a object int the main line topo
func (s *Service) DeleteMainLineObject(ctx *rest.Contexts) {
	objID := ctx.Request.PathParameter("bk_obj_id")

	// do with transaction
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		if err := s.Core.AssociationOperation().DeleteMainlineAssociation(ctx.Kit, objID); err != nil {
			blog.Errorf("DeleteMainlineAssociation failed, err: %+v, rid: %s", err, ctx.Kit.Rid)
			return ctx.Kit.CCError.CCError(common.CCErrTopoObjectDeleteFailed)
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)

}

// SearchMainLineObjectTopo search the main line topo
func (s *Service) SearchMainLineObjectTopo(ctx *rest.Contexts) {
	bizObj, err := s.Core.ObjectOperation().FindSingleObject(ctx.Kit, common.BKInnerObjIDApp)
	if nil != err {
		blog.Errorf("[api-asst] failed to find the biz object, error info is %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	// get biz model related mainline models (mainline relationship model)
	resp, err := s.Core.AssociationOperation().SearchMainlineAssociationTopo(ctx.Kit, bizObj)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(resp)
}

// SearchObjectByClassificationID search the object by classification ID
func (s *Service) SearchObjectByClassificationID(ctx *rest.Contexts) {
	bizObj, err := s.Core.ObjectOperation().FindSingleObject(ctx.Kit, ctx.Request.PathParameter("bk_obj_id"))
	if nil != err {
		blog.Errorf("[api-asst] failed to find the biz object, error info is %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	resp, err := s.Core.AssociationOperation().SearchMainlineAssociationTopo(ctx.Kit, bizObj)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(resp)
}

// SearchBusinessTopoWithStatistics calculate how many service instances on each topo instance node
func (s *Service) SearchBusinessTopoWithStatistics(ctx *rest.Contexts) {
	resp, err := s.searchBusinessTopo(ctx, true, true)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(resp)
}

func (s *Service) SearchBusinessTopo(ctx *rest.Contexts) {
	resp, err := s.searchBusinessTopo(ctx, false, false)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(resp)
}

// SearchBusinessTopo search the business topo
// withSortName 按拼音对名字排序，ui 专用
func (s *Service) searchBusinessTopo(ctx *rest.Contexts,
	withStatistics, withSortName bool) ([]*metadata.TopoInstRst, error) {
	id, err := strconv.ParseInt(ctx.Request.PathParameter("bk_biz_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-asst] failed to parse the path params id(%s), error info is %s , rid: %s", ctx.Request.PathParameter("app_id"), err.Error(), ctx.Kit.Rid)

		return nil, err
	}

	withDefault := false
	if len(ctx.Request.QueryParameter("with_default")) > 0 {
		withDefault = true
	}
	topoInstRst, err := s.Core.AssociationOperation().SearchMainlineAssociationInstTopo(ctx.Kit, common.BKInnerObjIDApp, id, withStatistics, withDefault)
	if err != nil {
		return nil, err
	}

	if withSortName {
		// sort before response,
		SortTopoInst(topoInstRst)
	}

	return topoInstRst, nil
}

// SearchBusinessTopoNodeHostAndServiceInstanceCount calculate how many service instances on each topo instance node amd how many hosts
func (s *Service) SearchBusinessTopoNodeHostAndServiceInstanceCount(ctx *rest.Contexts) {
	resp, err := s.searchBusinessTopoNodeHostAndServiceInstancesCount(ctx)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(resp)
}

func (s *Service) searchBusinessTopoNodeHostAndServiceInstancesCount(ctx *rest.Contexts) ([]*metadata.TopoInstNodeHostAndServiceInstCount, error) {
	id, err := strconv.ParseInt(ctx.Request.PathParameter("bk_biz_id"), 10, 64)
	if nil != err {
		blog.Errorf("[api-asst] failed to parse the path params id(%s), error info is %s , rid: %s", ctx.Request.PathParameter("app_id"), err.Error(), ctx.Kit.Rid)

		return nil, err
	}
	input := new(metadata.SearchBizTopoNodeHostAndServiceInstCountOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return nil, err
	}

	if len(input.Condition) > common.BKParamMaxLength {
		return nil, ctx.Kit.CCError.Error(common.CCErrorParamExceedMaxLength)
	}

	topoInstHostCountRst, err := s.Core.AssociationOperation().SearchTopoNodeHostAndServiceInstCount(ctx.Kit, id, input)
	if err != nil {
		return nil, err
	}

	return topoInstHostCountRst, nil
}

func SortTopoInst(instData []*metadata.TopoInstRst) {
	for _, data := range instData {
		instNameInGBK, _ := ioutil.ReadAll(transform.NewReader(bytes.NewReader([]byte(data.InstName)), simplifiedchinese.GBK.NewEncoder()))
		data.InstName = string(instNameInGBK)
	}

	sort.Slice(instData, func(i, j int) bool {
		return instData[i].InstName < instData[j].InstName
	})

	for _, data := range instData {
		instNameInUTF, _ := ioutil.ReadAll(transform.NewReader(bytes.NewReader([]byte(data.InstName)), simplifiedchinese.GBK.NewDecoder()))
		data.InstName = string(instNameInUTF)
	}

	for idx := range instData {
		SortTopoInst(instData[idx].Child)
	}
}

// SearchBriefBizTopo search brief topo
func (s *Service) SearchBriefBizTopo(ctx *rest.Contexts) {
	bizID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKAppIDField), 10, 64)
	if err != nil {
		blog.Errorf("SearchBriefBizTopo failed, parse bk_biz_id error, err: %s, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, "bk_biz_id"))
		return
	}

	input := new(metadata.SearchBriefBizTopoOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	setDetail, err := s.getSetDetailOfTopo(ctx, bizID, input)
	if err != nil {
		blog.Errorf("SearchBriefBizTopo failed, getSetDetailOfTopo err: %v, rid:%s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	if len(setDetail) == 0 {
		ctx.RespEntity([]interface{}{})
		return
	}

	moduleDetail, setModuleMap, err := s.getModuleInfoOfTopo(ctx, bizID, input)
	if err != nil {
		blog.Errorf("SearchBriefBizTopo failed, getModuleInfoOfTopo err: %v, rid:%s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	hostDetail, moduleHostMap, err := s.getHostInfoOfTopo(ctx, bizID, input)
	if err != nil {
		blog.Errorf("SearchBriefBizTopo failed, getHostInfoOfTopo err: %v, rid:%s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	// construct the final result
	bizTopo := s.constructBizTopo(setDetail, moduleDetail, hostDetail, setModuleMap, moduleHostMap)

	ctx.RespEntity(bizTopo)
}

// getSetDetailOfTopo get set detail of topo
func (s *Service) getSetDetailOfTopo(ctx *rest.Contexts, bizID int64, input *metadata.SearchBriefBizTopoOption) (map[int64]map[string]interface{}, errors.CCErrorCoder) {
	setDetail := make(map[int64]map[string]interface{})
	originSetFields := make(map[string]bool)
	for _, field := range input.SetFields {
		originSetFields[field] = true
	}
	input.SetFields = append(input.SetFields, common.BKSetIDField)

	pageSize := 2000
	start := 0
	hasNext := true
	param := &metadata.QueryCondition{
		Condition: map[string]interface{}{
			common.BKAppIDField: bizID,
		},
		Fields: input.SetFields,
		Page: metadata.BasePage{
			Start: start,
			Limit: pageSize,
		},
	}

	for hasNext {
		param.Page.Start = start
		setResult, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDSet, param)
		if nil != err {
			blog.Errorf("getSetDetailOfTopo failed, coreservice http ReadInstance fail, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
			return nil, ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
		}
		if !setResult.Result {
			blog.Errorf("getSetDetailOfTopo failed, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
			return nil, setResult.CCError()
		}

		if len(setResult.Data.Info) == 0 {
			break
		}

		for _, info := range setResult.Data.Info {
			setID, _ := info.Int64(common.BKSetIDField)
			if !originSetFields[common.BKDefaultField] {
				info.Remove(common.BKDefaultField)
			}
			setDetail[setID] = info
		}

		start += pageSize
		if len(setResult.Data.Info) < pageSize {
			hasNext = false
		}
	}

	return setDetail, nil
}

// getModuleInfoOfTopo get module info of topo
func (s *Service) getModuleInfoOfTopo(ctx *rest.Contexts, bizID int64, input *metadata.SearchBriefBizTopoOption) (
	map[int64]map[string]interface{}, map[int64][]int64, errors.CCErrorCoder) {
	//  get moduleDetail, setModuleMap
	moduleDetail := make(map[int64]map[string]interface{})
	setModuleMap := make(map[int64][]int64)

	originModuleFields := make(map[string]bool)
	for _, field := range input.ModuleFields {
		originModuleFields[field] = true
	}
	input.ModuleFields = append(input.ModuleFields, common.BKModuleIDField, common.BKSetIDField)

	pageSize := 2000
	start := 0
	hasNext := true
	param := &metadata.QueryCondition{
		Condition: map[string]interface{}{
			common.BKAppIDField: bizID,
		},
		Fields: input.ModuleFields,
		Page: metadata.BasePage{
			Start: start,
			Limit: pageSize,
		},
	}

	for hasNext {
		param.Page.Start = start
		moduleResult, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, param)
		if nil != err {
			blog.Errorf("getModuleInfoOfTopo failed, coreservice http ReadInstance fail, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
			return nil, nil, ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
		}
		if !moduleResult.Result {
			blog.Errorf("getModuleInfoOfTopo failed, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
			return nil, nil, moduleResult.CCError()
		}

		if len(moduleResult.Data.Info) == 0 {
			break
		}

		for _, info := range moduleResult.Data.Info {
			setID, _ := info.Int64(common.BKSetIDField)
			moduleID, _ := info.Int64(common.BKModuleIDField)
			setModuleMap[setID] = append(setModuleMap[setID], moduleID)

			if !originModuleFields[common.BKDefaultField] {
				info.Remove(common.BKDefaultField)
			}
			if !originModuleFields[common.BKSetIDField] {
				info.Remove(common.BKSetIDField)
			}
			moduleDetail[moduleID] = info
		}

		start += pageSize
		if len(moduleResult.Data.Info) < pageSize {
			hasNext = false
		}
	}

	return moduleDetail, setModuleMap, nil
}

// getHostInfoOfTopo get host info of topo
func (s *Service) getHostInfoOfTopo(ctx *rest.Contexts, bizID int64, input *metadata.SearchBriefBizTopoOption) (
	map[int64]map[string]interface{}, map[int64][]int64, errors.CCErrorCoder) {
	hostDetail := make(map[int64]map[string]interface{})
	moduleHostMap := make(map[int64][]int64)

	// get hostIDArr, moduleHostMap
	hostIDArr := make([]int64, 0)
	relationOption := &metadata.HostModuleRelationRequest{
		ApplicationID: bizID,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
		Fields: []string{common.BKModuleIDField, common.BKHostIDField},
	}
	hostModuleRelations, err := s.Engine.CoreAPI.CoreService().Host().GetHostModuleRelation(ctx.Kit.Ctx, ctx.Kit.Header, relationOption)
	if err != nil {
		blog.Errorf("getHostInfoOfTopo failed, option: %+v, err: %s, rid: %s", relationOption, err.Error(), ctx.Kit.Rid)
		return nil, nil, ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}
	for _, relation := range hostModuleRelations.Data.Info {
		hostIDArr = append(hostIDArr, relation.HostID)
		moduleHostMap[relation.ModuleID] = append(moduleHostMap[relation.ModuleID], relation.HostID)
	}

	// get hostDetail
	if len(hostIDArr) > 0 {
		pageSize := 2000
		start := 0
		hasNext := true
		input.HostFields = append(input.HostFields, common.BKHostIDField)
		param := &metadata.QueryCondition{
			Condition: map[string]interface{}{
				common.BKHostIDField: map[string]interface{}{
					common.BKDBIN: hostIDArr,
				},
			},
			Fields: input.HostFields,
			Page: metadata.BasePage{
				Start: start,
				Limit: pageSize,
			},
		}

		for hasNext {
			param.Page.Start = start
			hostResult, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDHost, param)
			if nil != err {
				blog.Errorf("getHostInfoOfTopo failed, coreservice http ReadInstance fail, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
				return nil, nil, ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
			}
			if !hostResult.Result {
				blog.Errorf("getHostInfoOfTopo failed, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
				return nil, nil, hostResult.CCError()
			}

			if len(hostResult.Data.Info) == 0 {
				break
			}

			for _, info := range hostResult.Data.Info {
				hostID, _ := info.Int64(common.BKHostIDField)
				hostDetail[hostID] = info
			}

			start += pageSize
			if len(hostResult.Data.Info) < pageSize {
				hasNext = false
			}
		}
	}

	return hostDetail, moduleHostMap, nil
}

// constructBizTopo construct biz topo
func (s *Service) constructBizTopo(setDetail, moduleDetail, hostDetail map[int64]map[string]interface{}, setModuleMap,
	moduleHostMap map[int64][]int64) []*metadata.SetTopo {
	bizTopo := make([]*metadata.SetTopo, 0)
	for setID, set := range setDetail {
		setTopo := new(metadata.SetTopo)
		setTopo.Set = set
		moduleTopos := make([]*metadata.ModuleTopo, 0)
		for _, moduleID := range setModuleMap[setID] {
			moduleTopo := new(metadata.ModuleTopo)
			moduleTopo.Module = moduleDetail[moduleID]
			hosts := make([]map[string]interface{}, 0)
			for _, hostID := range moduleHostMap[moduleID] {
				hosts = append(hosts, hostDetail[hostID])
			}
			moduleTopo.Hosts = hosts
			moduleTopos = append(moduleTopos, moduleTopo)
		}
		setTopo.ModuleTopos = moduleTopos
		bizTopo = append(bizTopo, setTopo)
	}
	return bizTopo
}

func (s *Service) SearchAssociationType(ctx *rest.Contexts) {
	request := &metadata.SearchAssociationTypeRequest{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}
	if request.Condition == nil {
		request.Condition = make(map[string]interface{}, 0)
	}

	ret, err := s.Core.AssociationOperation().SearchType(ctx.Kit, request)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	if ret.Code != 0 {
		ctx.RespAutoError(ctx.Kit.CCError.New(ret.Code, ret.ErrMsg))
		return
	}

	ctx.RespEntity(ret.Data)
}

func (s *Service) SearchObjectAssocWithAssocKindList(ctx *rest.Contexts) {

	ids := new(metadata.AssociationKindIDs)
	if err := ctx.DecodeInto(ids); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommParamsInvalid))
		return
	}

	resp, err := s.Core.AssociationOperation().SearchObjectAssocWithAssocKindList(ctx.Kit, ids.AsstIDs)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(resp)
}

func (s *Service) CreateAssociationType(ctx *rest.Contexts) {
	request := &metadata.AssociationKind{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}

	var ret *metadata.CreateAssociationTypeResult
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		ret, err = s.Core.AssociationOperation().CreateType(ctx.Kit, request)
		if err != nil {
			return err
		}

		if ret.Code != 0 {
			return ctx.Kit.CCError.New(ret.Code, ret.ErrMsg)
		}

		// register association type resource creator action to iam
		if auth.EnableAuthorize() {
			iamInstance := metadata.IamInstanceWithCreator{
				Type:    string(iam.SysAssociationType),
				ID:      strconv.FormatInt(ret.Data.ID, 10),
				Name:    request.AssociationKindName,
				Creator: ctx.Kit.User,
			}
			_, err = s.AuthManager.Authorizer.RegisterResourceCreatorAction(ctx.Kit.Ctx, ctx.Kit.Header, iamInstance)
			if err != nil {
				blog.Errorf("register created association type to iam failed, err: %v, rid: %s", err, ctx.Kit.Rid)
				return err
			}
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(ret.Data)
}

func (s *Service) UpdateAssociationType(ctx *rest.Contexts) {
	request := &metadata.UpdateAssociationTypeRequest{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}

	asstTypeID, err := strconv.ParseInt(ctx.Request.PathParameter("id"), 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}

	var ret *metadata.UpdateAssociationTypeResult
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		ret, err = s.Core.AssociationOperation().UpdateType(ctx.Kit, asstTypeID, request)
		if err != nil {
			return err
		}

		if ret.Code != 0 {
			return ctx.Kit.CCError.New(ret.Code, ret.ErrMsg)
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(ret.Data)
}

func (s *Service) DeleteAssociationType(ctx *rest.Contexts) {
	asstTypeID, err := strconv.ParseInt(ctx.Request.PathParameter("id"), 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}

	var ret *metadata.DeleteAssociationTypeResult
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		ret, err = s.Core.AssociationOperation().DeleteType(ctx.Kit, asstTypeID)
		if err != nil {
			return err
		}

		if ret.Code != 0 {
			return ctx.Kit.CCError.New(ret.Code, ret.ErrMsg)
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(ret.Data)
}

func (s *Service) SearchAssociationInst(ctx *rest.Contexts) {
	request := &metadata.SearchAssociationInstRequest{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}

	ctx.SetReadPreference(common.SecondaryPreferredMode)
	ret, err := s.Core.AssociationOperation().SearchInst(ctx.Kit, request)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	if ret.Code != 0 {
		ctx.RespAutoError(ctx.Kit.CCError.New(ret.Code, ret.ErrMsg))
		return
	}

	ctx.RespEntity(ret.Data)
}

//Search all associations of certain model instance,by regarding the instance as both Association source and Association target.
func (s *Service) SearchAssociationRelatedInst(ctx *rest.Contexts) {
	request := &metadata.SearchAssociationRelatedInstRequest{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommParamsInvalid, err.Error()))
		return
	}
	//check condition
	if request.Condition.InstID == 0 || request.Condition.ObjectID == "" {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommParamsInvalid, "'bk_inst_id' and 'bk_obj_id' should not be empty."))
		return
	}
	//check fields,if there's none param,return err.
	if len(request.Fields) == 0 {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommParamsInvalid, "there should be at least one param in 'fields'."))
		return
	}
	//Use id as sort parameters
	request.Page.Sort = common.BKFieldID
	//check Maximum limit
	if request.Page.Limit > common.BKMaxInstanceLimit {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommParamsInvalid, "The maximum limit should be less than 500."))
		return
	}

	ret, err := s.Core.AssociationOperation().SearchAssociationRelatedInst(ctx.Kit, request)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	if err := ret.CCError(); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(ret.Code, ret.ErrMsg))
		return
	}

	ctx.RespEntity(ret.Data)
}

func (s *Service) CreateAssociationInst(ctx *rest.Contexts) {
	request := &metadata.CreateAssociationInstRequest{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}

	var ret *metadata.CreateAssociationInstResult
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		ret, err = s.Core.AssociationOperation().CreateInst(ctx.Kit, request)
		if err != nil {
			return err
		}

		if ret.Code != 0 {
			return ctx.Kit.CCError.New(ret.Code, ret.ErrMsg)
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(ret.Data)
}

func (s *Service) DeleteAssociationInst(ctx *rest.Contexts) {
	id, err := strconv.ParseInt(ctx.Request.PathParameter("association_id"), 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommParamsIsInvalid))
		return
	}

	var ret *metadata.DeleteAssociationInstResult
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		idList := []int64{id}
		// bkObjID has no actual effect here for now.
		bkObjID := ""
		ret, err = s.Core.AssociationOperation().DeleteInst(ctx.Kit, idList, bkObjID)
		if err != nil {
			return err
		}

		if ret.Code != 0 {
			return ctx.Kit.CCError.New(ret.Code, ret.ErrMsg)
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(ret.Data)
}

func (s *Service) DeleteAssociationInstBatch(ctx *rest.Contexts) {
	request := &metadata.DeleteAssociationInstBatchRequest{}
	if err := ctx.DecodeInto(request); err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.New(common.CCErrCommParamsInvalid, err.Error()))
		return
	}
	if len(request.ID) == 0 {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommHTTPInputInvalid))
		return
	}
	if len(request.ID) > common.BKMaxInstanceLimit {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommPageLimitIsExceeded, "The number of ID should be less than 500."))
		return
	}
	if request.ObjectID == "" {
		ctx.RespAutoError(ctx.Kit.CCError.Errorf(common.CCErrCommHTTPInputInvalid))
		return
	}

	result := &metadata.DeleteAssociationInstBatchResult{}
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var ret *metadata.DeleteAssociationInstResult
		var err error
		ret, err = s.Core.AssociationOperation().DeleteInst(ctx.Kit, request.ID, request.ObjectID)
		if err != nil {
			return err
		}
		if err = ret.CCError(); err != nil {
			return err
		}
		result.Data = len(request.ID)
		return nil
	})
	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}

	ctx.RespEntity(result.Data)
}

func (s *Service) SearchTopoPath(ctx *rest.Contexts) {
	rid := ctx.Kit.Rid

	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if nil != err {
		blog.Errorf("SearchTopoPath failed, bizIDStr: %s, err: %s, rid: %s", bizIDStr, err.Error(), rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	input := metadata.FindTopoPathRequest{}
	if err := ctx.DecodeInto(&input); err != nil {
		ctx.RespAutoError(err)
		return
	}
	if len(input.Nodes) == 0 {
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPBodyEmpty))
		return
	}

	topoRoot, err := s.Engine.CoreAPI.CoreService().Mainline().SearchMainlineInstanceTopo(ctx.Kit.Ctx, ctx.Kit.Header, bizID, false)
	if err != nil {
		blog.Errorf("SearchTopoPath failed, SearchMainlineInstanceTopo failed, bizID:%d, err:%s, rid:%s", bizID, err.Error(), rid)
		ctx.RespAutoError(err)
		return
	}
	result := metadata.TopoPathResult{}
	for _, node := range input.Nodes {
		topoPath := topoRoot.TraversalFindNode(node.ObjectID, node.InstanceID)
		path := make([]*metadata.TopoInstanceNodeSimplify, 0)
		for _, item := range topoPath {
			simplify := item.ToSimplify()
			path = append(path, simplify)
		}
		nodeTopoPath := metadata.NodeTopoPath{
			BizID: bizID,
			Node:  node,
			Path:  path,
		}
		result.Nodes = append(result.Nodes, nodeTopoPath)
	}

	ctx.RespEntity(result)
}

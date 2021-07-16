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
	"strconv"

	"configcenter/src/ac/iam"
	"configcenter/src/common"
	"configcenter/src/common/auth"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

func (s *Service) CreateSetTemplate(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	option := metadata.CreateSetTemplateOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	if len(option.ServiceTemplateIDs) == 0 {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, "service_template_ids"))
		return
	}

	var setTemplate metadata.SetTemplate
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		setTemplate, err = s.Engine.CoreAPI.CoreService().SetTemplate().CreateSetTemplate(ctx.Kit.Ctx, ctx.Kit.Header, bizID, option)
		if err != nil {
			blog.Errorf("CreateSetTemplate failed, core service create failed, bizID: %d, option: %+v, err: %+v, rid: %s", bizID, option, err, ctx.Kit.Rid)
			return err
		}

		// register set template resource creator action to iam
		if auth.EnableAuthorize() {
			iamInstance := metadata.IamInstanceWithCreator{
				Type:    string(iam.BizSetTemplate),
				ID:      strconv.FormatInt(setTemplate.ID, 10),
				Name:    setTemplate.Name,
				Creator: ctx.Kit.User,
			}
			_, err = s.AuthManager.Authorizer.RegisterResourceCreatorAction(ctx.Kit.Ctx, ctx.Kit.Header, iamInstance)
			if err != nil {
				blog.Errorf("register created set template to iam failed, err: %v, rid: %s", err, ctx.Kit.Rid)
				return err
			}
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(setTemplate)
}

func (s *Service) UpdateSetTemplate(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	option := metadata.UpdateSetTemplateOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	var setTemplate metadata.SetTemplate
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		setTemplate, err = s.Engine.CoreAPI.CoreService().SetTemplate().UpdateSetTemplate(ctx.Kit.Ctx, ctx.Kit.Header, bizID, setTemplateID, option)
		if err != nil {
			blog.Errorf("UpdateSetTemplate failed, do core service update failed, bizID: %d, option: %+v, err: %+v, rid: %s", bizID, option, err, ctx.Kit.Rid)
			return err
		}

		filter := &metadata.DistinctFieldOption{
			TableName: common.GetInstTableName(common.BKInnerObjIDSet),
			Field:     common.BKSetIDField,
			Filter: map[string]interface{}{
				common.BKSetTemplateIDField: setTemplateID,
			},
		}

		setIDs, err := s.Engine.CoreAPI.CoreService().Common().GetDistinctField(ctx.Kit.Ctx, ctx.Kit.Header, filter)
		if err != nil {
			blog.Errorf("get set id failed, err: %+v, rid: %s", err, ctx.Kit.Rid)
			return err
		}

		if len(setIDs) == 0 {
			blog.Errorf("get set id failed, GetDistinctField return is empty, rid: %s", ctx.Kit.Rid)
			return nil
		}

		setID, err := util.SliceInterfaceToInt64(setIDs)
		if err != nil {
			blog.Errorf("set id is not int, err: %s, rid: %s", err.Error(), ctx.Kit.Rid)
			return err
		}

		if _, err := s.Core.SetTemplateOperation().UpdateSetSyncStatus(ctx.Kit, setTemplateID, setID); err != nil {
			blog.Errorf("update set sync status failed, setID: %d, err: %+v, rid: %s", setID, err, ctx.Kit.Rid)
			return err
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(setTemplate)
}

func (s *Service) DeleteSetTemplate(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	option := metadata.DeleteSetTemplateOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		if err := s.Engine.CoreAPI.CoreService().SetTemplate().DeleteSetTemplate(ctx.Kit.Ctx, ctx.Kit.Header, bizID, option); err != nil {
			blog.Errorf("DeleteSetTemplate failed, do core service update failed, bizID: %d, option: %+v, err: %+v, rid: %s", bizID, option, err, ctx.Kit.Rid)
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

func (s *Service) GetSetTemplate(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	setTemplate, err := s.Engine.CoreAPI.CoreService().SetTemplate().GetSetTemplate(ctx.Kit.Ctx, ctx.Kit.Header, bizID, setTemplateID)
	if err != nil {
		blog.Errorf("GetSetTemplate failed, do core service get failed, bizID: %d, setTemplateID: %d, err: %+v, rid: %s", bizID, setTemplateID, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(setTemplate)
}

func (s *Service) ListSetTemplate(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	option := metadata.ListSetTemplateOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}
	// set default value
	if option.Page.Limit == 0 {
		option.Page.Limit = common.BKDefaultLimit
	}

	setTemplate, err := s.Engine.CoreAPI.CoreService().SetTemplate().ListSetTemplate(ctx.Kit.Ctx, ctx.Kit.Header, bizID, option)
	if err != nil {
		blog.Errorf("ListSetTemplate failed, do core service ListSetTemplate failed, bizID: %d, option: %+v, err: %+v, rid: %s", bizID, option, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(setTemplate)
}

func (s *Service) ListSetTemplateWeb(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	listOption := metadata.ListSetTemplateOption{}
	if err := ctx.DecodeInto(&listOption); err != nil {
		ctx.RespAutoError(err)
		return
	}
	if listOption.Page.Limit == 0 {
		listOption.Page.Limit = common.BKNoLimit
	}

	listResult, err := s.Engine.CoreAPI.CoreService().SetTemplate().ListSetTemplate(ctx.Kit.Ctx, ctx.Kit.Header, bizID, listOption)
	if err != nil {
		blog.Errorf("ListSetTemplate failed, do core service ListSetTemplate failed, bizID: %d, option: %+v, err: %+v, rid: %s", bizID, listOption, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if listResult == nil {
		ctx.RespEntity(nil)
		return
	}

	// count template instances
	setTemplateIDs := make([]int64, 0)
	for _, item := range listResult.Info {
		setTemplateIDs = append(setTemplateIDs, item.ID)
	}
	option := metadata.CountSetTplInstOption{
		SetTemplateIDs: setTemplateIDs,
	}
	setTplInstCount, err := s.Engine.CoreAPI.CoreService().SetTemplate().CountSetTplInstances(ctx.Kit.Ctx, ctx.Kit.Header, bizID, option)
	if err != nil {
		blog.Errorf("ListSetTemplateWeb failed, CountSetTplInstances failed, bizID: %d, option: %+v, err: %s, rid: %s", bizID, option, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	result := metadata.MultipleSetTemplateWithStatisticsResult{
		Count: listResult.Count,
	}
	for _, setTemplate := range listResult.Info {
		setInstanceCount, exist := setTplInstCount[setTemplate.ID]
		if exist == false {
			setInstanceCount = 0
		}
		result.Info = append(result.Info, metadata.SetTemplateWithStatistics{
			SetInstanceCount: setInstanceCount,
			SetTemplate:      setTemplate,
		})
	}
	ctx.RespEntity(result)
}

func (s *Service) ListSetTplRelatedSvcTpl(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	serviceTemplates, err := s.Engine.CoreAPI.CoreService().SetTemplate().ListSetTplRelatedSvcTpl(ctx.Kit.Ctx, ctx.Kit.Header, bizID, setTemplateID)
	if err != nil {
		blog.Errorf("ListSetTemplateRelatedServiceTemplate failed, ListSetTplRelatedSvcTpl failed, bizID: %d, setTemplateID: %+v, err: %+v, rid: %s", bizID, setTemplateID, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(serviceTemplates)
}

func (s *Service) ListSetTplRelatedSvcTplWithStatistics(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	serviceTemplates, err := s.Engine.CoreAPI.CoreService().SetTemplate().ListSetTplRelatedSvcTpl(ctx.Kit.Ctx, ctx.Kit.Header, bizID, setTemplateID)
	if err != nil {
		blog.Errorf("ListSetTplRelatedSvcTplWithStatistics failed, do core service list failed, bizID: %d, setTemplateID: %+v, err: %+v, rid: %s", bizID, setTemplateID, err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	serviceTemplateIDs := make([]int64, 0)
	for _, item := range serviceTemplates {
		serviceTemplateIDs = append(serviceTemplateIDs, item.ID)
	}
	moduleFilter := &metadata.QueryCondition{
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
		Condition: map[string]interface{}{
			common.BKServiceTemplateIDField: map[string]interface{}{
				common.BKDBIN: serviceTemplateIDs,
			},
			common.BKSetTemplateIDField: setTemplateID,
		},
	}

	moduleResult := new(metadata.ResponseModuleInstance)
	if err := s.Engine.CoreAPI.CoreService().Instance().ReadInstanceStruct(ctx.Kit.Ctx, ctx.Kit.Header,
		common.BKInnerObjIDModule, moduleFilter, moduleResult); err != nil {
		blog.ErrorJSON("ListSetTplRelatedSvcTplWithStatistics failed, ReadInstance of module http failed, "+
			"option: %s, err: %s, rid: %s", moduleFilter, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	if ccErr := moduleResult.CCError(); ccErr != nil {
		blog.Errorf("ListSetTplRelatedSvcTplWithStatistics failed, ReadInstance of module failed, filter: %s, result: %s, rid: %s", moduleFilter, moduleResult, ctx.Kit.Rid)
		ctx.RespAutoError(ccErr)
		return
	}
	moduleIDs := make([]int64, 0)
	svcTpl2Modules := make(map[int64][]metadata.ModuleInst)
	// map[module]service_template_id
	moduleIDSvcTplID := make(map[int64]int64, 0)
	for _, module := range moduleResult.Data.Info {
		moduleIDSvcTplID[module.ModuleID] = module.ServiceTemplateID
		svcTpl2Modules[module.ServiceTemplateID] = append(svcTpl2Modules[module.ServiceTemplateID], module)
		moduleIDs = append(moduleIDs, module.ModuleID)
	}

	// host module relations
	relationOption := metadata.HostModuleRelationRequest{
		ApplicationID: bizID,
		ModuleIDArr:   moduleIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
		Fields: []string{common.BKModuleIDField, common.BKHostIDField},
	}
	relationResult, err := s.Engine.CoreAPI.CoreService().Host().GetHostModuleRelation(ctx.Kit.Ctx, ctx.Kit.Header, &relationOption)
	if err != nil {
		blog.Errorf("ListSetTplRelatedSvcTplWithStatistics failed, GetHostModuleRelation http failed, option: %s, err: %s, rid: %s", relationOption, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if ccErr := relationResult.CCError(); ccErr != nil {
		blog.Errorf("ListSetTplRelatedSvcTplWithStatistics failed, GetHostModuleRelation failed, option: %s, result: %s, rid: %s", relationOption, relationResult, ctx.Kit.Rid)
		ctx.RespAutoError(ccErr)
		return
	}

	// module hosts
	svcTplIDHostIDs := make(map[int64][]int64)
	for _, item := range relationResult.Data.Info {
		if svcTplID, ok := moduleIDSvcTplID[item.ModuleID]; ok {
			svcTplIDHostIDs[svcTplID] = append(svcTplIDHostIDs[svcTplID], item.HostID)
		}
	}

	type ServiceTemplateWithModuleInfo struct {
		ServiceTemplate metadata.ServiceTemplate `json:"service_template"`
		HostCount       int                      `json:"host_count"`
		Modules         []metadata.ModuleInst    `json:"modules"`
	}
	result := make([]ServiceTemplateWithModuleInfo, 0)
	for _, svcTpl := range serviceTemplates {
		info := ServiceTemplateWithModuleInfo{
			ServiceTemplate: svcTpl,
		}
		modules, ok := svcTpl2Modules[svcTpl.ID]
		if ok == false {
			result = append(result, info)
			continue
		}
		info.Modules = modules
		info.HostCount = len(util.IntArrayUnique(svcTplIDHostIDs[svcTpl.ID]))
		result = append(result, info)
	}

	ctx.RespEntity(result)
}

// ListSetTplRelatedSets get SetTemplate related sets
func (s *Service) ListSetTplRelatedSets(kit *rest.Kit, bizID int64, setTemplateID int64, option metadata.ListSetByTemplateOption) (*metadata.QueryConditionResult, error) {
	filter := map[string]interface{}{
		common.BKAppIDField:         bizID,
		common.BKSetTemplateIDField: setTemplateID,
	}
	if option.SetIDs != nil {
		filter[common.BKSetIDField] = map[string]interface{}{
			common.BKDBIN: option.SetIDs,
		}
	}
	qc := &metadata.QueryCondition{
		Page:      option.Page,
		Condition: filter,
	}
	return s.Engine.CoreAPI.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, common.BKInnerObjIDSet, qc)
}

// ListSetTplRelatedSetsWeb get SetTemplate related sets, just for web
func (s *Service) ListSetTplRelatedSetsWeb(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}
	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	option := metadata.ListSetByTemplateOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	response, err := s.ListSetTplRelatedSets(ctx.Kit, bizID, setTemplateID, option)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}
	setInstanceResult := response.Data

	topoTree, err := s.Engine.CoreAPI.CoreService().Mainline().SearchMainlineInstanceTopo(ctx.Kit.Ctx, ctx.Kit.Header, bizID, false)
	if err != nil {
		blog.Errorf("ListSetTplRelatedSetsWeb failed, bizID: %d, err: %s, rid: %s", bizID, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	setIDs := make([]int64, 0)
	for index := range setInstanceResult.Info {
		set := metadata.SetInst{}
		if err := mapstr.DecodeFromMapStr(&set, setInstanceResult.Info[index]); err != nil {
			ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommJSONUnmarshalFailed))
			return
		}
		setIDs = append(setIDs, set.SetID)

		setPath := topoTree.TraversalFindNode(common.BKInnerObjIDSet, set.SetID)
		topoPath := make([]metadata.TopoInstanceNodeSimplify, 0)
		for _, pathNode := range setPath {
			nodeSimplify := metadata.TopoInstanceNodeSimplify{
				ObjectID:     pathNode.ObjectID,
				InstanceID:   pathNode.InstanceID,
				InstanceName: pathNode.InstanceName,
			}
			topoPath = append(topoPath, nodeSimplify)
		}
		setInstanceResult.Info[index]["topo_path"] = topoPath
	}

	// fill with host count
	filter := &metadata.HostModuleRelationRequest{
		ApplicationID: bizID,
		SetIDArr:      setIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
		Fields: []string{common.BKSetIDField, common.BKHostIDField},
	}
	relations, err := s.Engine.CoreAPI.CoreService().Host().GetHostModuleRelation(ctx.Kit.Ctx, ctx.Kit.Header, filter)
	if err != nil {
		blog.ErrorJSON("SearchMainlineInstanceTopo failed, GetHostModuleRelation failed, filter: %s, err: %s, rid: %s", filter, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if relations.Result == false || relations.Code != 0 {
		blog.ErrorJSON("SearchMainlineInstanceTopo failed, GetHostModuleRelation return false, filter: %s, result: %s, rid: %s", filter, relations, ctx.Kit.Rid)
		ctx.RespAutoError(errors.NewCCError(relations.Code, relations.ErrMsg))
		return
	}
	set2Hosts := make(map[int64][]int64)
	for _, relation := range relations.Data.Info {
		if _, ok := set2Hosts[relation.SetID]; ok == false {
			set2Hosts[relation.SetID] = make([]int64, 0)
		}
		set2Hosts[relation.SetID] = append(set2Hosts[relation.SetID], relation.HostID)
	}
	for setID := range set2Hosts {
		set2Hosts[setID] = util.IntArrayUnique(set2Hosts[setID])
	}

	for index := range setInstanceResult.Info {
		set := metadata.SetInst{}
		if err := mapstr.DecodeFromMapStr(&set, setInstanceResult.Info[index]); err != nil {
			ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommJSONUnmarshalFailed))
			return
		}
		hostCount := 0
		if _, ok := set2Hosts[set.SetID]; ok == true {
			hostCount = len(set2Hosts[set.SetID])
		}
		setInstanceResult.Info[index]["host_count"] = hostCount
	}

	ctx.RespEntity(setInstanceResult)
}

func (s *Service) DiffSetTplWithInst(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	option := metadata.DiffSetTplWithInstOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	setDiffs, err := s.Core.SetTemplateOperation().DiffSetTplWithInst(ctx.Kit, bizID, setTemplateID, option)
	if err != nil {
		blog.Errorf("DiffSetTplWithInst failed, operation failed, bizID: %d, setTemplateID: %d, option: %+v err: %s, rid: %s", bizID, setTemplateID, option, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	moduleIDs := make([]int64, 0)
	for _, setDiff := range setDiffs {
		for _, moduleDiff := range setDiff.ModuleDiffs {
			if moduleDiff.ModuleID == 0 {
				continue
			}
			moduleIDs = append(moduleIDs, moduleDiff.ModuleID)
		}
	}

	result := metadata.SetTplDiffResult{
		Difference:      setDiffs,
		ModuleHostCount: make(map[int64]int64),
	}

	if len(moduleIDs) > 0 {
		relationOption := &metadata.HostModuleRelationRequest{
			ApplicationID: bizID,
			ModuleIDArr:   moduleIDs,
			Page: metadata.BasePage{
				Limit: common.BKNoLimit,
			},
			Fields: []string{common.BKModuleIDField},
		}
		relationResult, err := s.Engine.CoreAPI.CoreService().Host().GetHostModuleRelation(ctx.Kit.Ctx, ctx.Kit.Header, relationOption)
		if err != nil {
			ctx.RespAutoError(err)
			return
		}
		moduleHostsCount := make(map[int64]int64)
		for _, item := range relationResult.Data.Info {
			if _, exist := moduleHostsCount[item.ModuleID]; exist == false {
				moduleHostsCount[item.ModuleID] = 0
			}
			moduleHostsCount[item.ModuleID] += 1
		}
		for _, moduleID := range moduleIDs {
			if _, exist := moduleHostsCount[moduleID]; exist == false {
				moduleHostsCount[moduleID] = 0
			}
		}
		result.ModuleHostCount = moduleHostsCount
	}

	ctx.RespEntity(result)
}

func (s *Service) SyncSetTplToInst(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	option := metadata.SyncSetTplToInstOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	taskCond := metadata.ListAPITaskDetail{
		SetID: option.SetIDs,
		Fields: []string{
			common.BKStatusField,
			common.MetaDataSynchronizeFlagField,
			common.BKInstIDField,
		},
	}
	// NOTE: 如下处理不能杜绝所有发提交任务, 可通过前端防双击的方式限制绝大部分情况
	setSyncStatus, err := s.Core.SetTemplateOperation().GetLatestSyncTaskDetail(ctx.Kit, taskCond)
	if err != nil {
		blog.Errorf("SyncSetTplToInst failed, getSetSyncStatus failed, setIDs: %+v, err: %s, rid: %s", option.SetIDs, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	for _, setID := range option.SetIDs {
		setStatus, exist := setSyncStatus[setID]
		if !exist {
			continue
		}
		if setStatus.Status.IsFinished() == false {
			ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrorTopoSyncModuleTaskIsRunning))
			return
		}
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		if err := s.Core.SetTemplateOperation().SyncSetTplToInst(ctx.Kit, bizID, setTemplateID, option); err != nil {
			blog.Errorf("SyncSetTplToInst failed, operation failed, bizID: %d, setTemplateID: %d, option: %+v err: %s, rid: %s", bizID, setTemplateID, option, err.Error(), ctx.Kit.Rid)
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

func (s *Service) GetSetSyncDetails(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	option := metadata.SetSyncStatusOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}
	if option.SetIDs == nil {
		filter := &metadata.QueryCondition{
			Page: metadata.BasePage{
				Limit: common.BKNoLimit,
			},
			Condition: mapstr.MapStr(map[string]interface{}{
				common.BKAppIDField:         bizID,
				common.BKSetTemplateIDField: setTemplateID,
			}),
		}
		setInstanceResult, err := s.Engine.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDSet, filter)
		if err != nil {
			blog.Errorf("GetSetSyncStatus failed, get template related set failed, err: %+v, rid: %s", err, ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}
		setIDs := make([]int64, 0)
		for _, inst := range setInstanceResult.Data.Info {
			setID, err := inst.Int64(common.BKSetIDField)
			if err != nil {
				blog.Errorf("GetSetSyncStatus failed, get template related set failed, err: %+v, rid: %s", err, ctx.Kit.Rid)
				ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommParseDBFailed))
				return
			}
			setIDs = append(setIDs, setID)
		}
		option.SetIDs = setIDs
	}

	taskCond := metadata.ListAPITaskDetail{
		SetID: option.SetIDs,
		Fields: []string{
			common.BKStatusField,
			common.MetaDataSynchronizeFlagField,
			common.BKInstIDField,
			"detail.status",
			"detail.data.module_diff.bk_module_name",
			"detail.response.baseresp.errmsg",
		},
	}

	taskDetail, err := s.Core.SetTemplateOperation().GetLatestSyncTaskDetail(ctx.Kit, taskCond)
	if err != nil {
		blog.Errorf("get the latest task detail failed, err: %s, rid: %s", err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(taskDetail)
	return
}

func (s *Service) ListSetTemplateSyncHistory(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	option := metadata.ListSetTemplateSyncStatusOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	result, err := s.Engine.CoreAPI.CoreService().SetTemplate().ListSetTemplateSyncHistory(ctx.Kit.Ctx, ctx.Kit.Header, bizID, option)
	if err != nil {
		blog.ErrorJSON("ListSetTemplateSyncHistory failed, core service search failed, option: %s, err: %s, rid: %s", option, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(result)
	return
}

func (s *Service) ListSetTemplateSyncStatus(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	option := metadata.ListSetTemplateSyncStatusOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	result, err := s.Core.SetTemplateOperation().ListSetTemplateSyncStatus(ctx.Kit, bizID, option)
	if err != nil {
		blog.ErrorJSON("ListSetTemplateSyncStatus failed, core service search failed, option: %s, err: %s, rid: %s", option, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(result)
	return
}

func (s *Service) CheckSetInstUpdateToDateStatus(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	setTemplateIDStr := ctx.Request.PathParameter(common.BKSetTemplateIDField)
	setTemplateID, err := strconv.ParseInt(setTemplateIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKSetTemplateIDField))
		return
	}

	result, err := s.Core.SetTemplateOperation().CheckSetInstUpdateToDateStatus(ctx.Kit, bizID, setTemplateID)
	if err != nil {
		blog.ErrorJSON("CheckSetInstUpdateToDateStatus failed, call core implement failed, bizID: %d, setTemplateID: %d, err: %s, rid: %s", bizID, setTemplateID, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(result)
	return
}

func (s *Service) BatchCheckSetInstUpdateToDateStatus(ctx *rest.Contexts) {
	bizIDStr := ctx.Request.PathParameter(common.BKAppIDField)
	bizID, err := strconv.ParseInt(bizIDStr, 10, 64)
	if err != nil {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField))
		return
	}

	option := metadata.BatchCheckSetInstUpdateToDateStatusOption{}
	if err := ctx.DecodeInto(&option); err != nil {
		ctx.RespAutoError(err)
		return
	}

	batchResult := make([]metadata.SetTemplateUpdateToDateStatus, 0)
	for _, setTemplateID := range option.SetTemplateIDs {
		oneResult, err := s.Core.SetTemplateOperation().CheckSetInstUpdateToDateStatus(ctx.Kit, bizID, setTemplateID)
		if err != nil {
			blog.ErrorJSON("BatchCheckSetInstUpdateToDateStatus failed, CheckSetInstUpdateToDateStatus failed, bizID: %d, setTemplateID: %d, err: %s, rid: %s", bizID, setTemplateID, err.Error(), ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}
		batchResult = append(batchResult, *oneResult)
	}
	ctx.RespEntity(batchResult)
}

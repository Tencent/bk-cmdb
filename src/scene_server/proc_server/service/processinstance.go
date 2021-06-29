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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/json"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

func (ps *ProcServer) CreateProcessInstances(ctx *rest.Contexts) {
	input := new(metadata.CreateRawProcessInstanceInput)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	var processIDs []int64
	txnErr := ps.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		processIDs, err = ps.createProcessInstances(ctx, input)
		if err != nil {
			blog.Errorf("create process instance failed, serviceInstanceID: %d, input: %+v, err: %+v", input.ServiceInstanceID, input, err)
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(processIDs)
}

func (ps *ProcServer) createProcessInstances(ctx *rest.Contexts, input *metadata.CreateRawProcessInstanceInput) ([]int64, errors.CCErrorCoder) {
	serviceInstance, err := ps.CoreAPI.CoreService().Process().GetServiceInstance(ctx.Kit.Ctx, ctx.Kit.Header, input.ServiceInstanceID)
	if err != nil {
		blog.Errorf("create process instance failed, get service instance by id failed, serviceInstanceID: %d, err: %v, rid: %s", input.ServiceInstanceID, err, ctx.Kit.Rid)
		return nil, err
	}
	if serviceInstance.BizID != input.BizID {
		blog.Errorf("create process instance with raw, biz id from input not equal with service instance, rid: %s", ctx.Kit.Rid)
		return nil, ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKAppIDField)
	}
	if serviceInstance.ServiceTemplateID != common.ServiceTemplateIDNotSet {
		blog.Errorf("create process instance failed, create process instance on service instance initialized by template forbidden, serviceInstanceID: %d, err: %v, rid: %s", input.ServiceInstanceID, err, ctx.Kit.Rid)
		return nil, ctx.Kit.CCError.CCError(common.CCErrProcEditProcessInstanceCreateByTemplateForbidden)
	}

	processIDs := make([]int64, 0)
	processDatas := make([]map[string]interface{}, len(input.Processes))
	for idx, item := range input.Processes {
		now := time.Now()
		item.ProcessData[common.BKProcessIDField] = int64(0)
		item.ProcessData[common.BKAppIDField] = input.BizID
		item.ProcessData[common.BkSupplierAccount] = ctx.Kit.SupplierAccount
		item.ProcessData[common.CreateTimeField] = now
		item.ProcessData[common.LastTimeField] = now

		processDatas[idx] = item.ProcessData
	}

	if err := ps.validateManyRawInstanceUnique(ctx.Kit, serviceInstance, processDatas); err != nil {
		blog.Errorf("create process instance failed, validateManyRawInstanceUnique err:%v, rid: %s", err, ctx.Kit.Rid)
		return nil, err
	}

	processIDs, err = ps.Logic.CreateProcessInstances(ctx.Kit, processDatas)
	if err != nil {
		blog.Errorf("create process instance failed, create process failed, serviceInstanceID: %d, processDatas: %+v, err: %v, rid: %s", input.ServiceInstanceID, processDatas, err, ctx.Kit.Rid)
		return nil, err
	}

	relations := make([]*metadata.ProcessInstanceRelation, len(processIDs))
	for idx, processID := range processIDs {
		relation := &metadata.ProcessInstanceRelation{
			BizID:             input.BizID,
			ProcessID:         processID,
			ProcessTemplateID: common.ServiceTemplateIDNotSet,
			ServiceInstanceID: serviceInstance.ID,
			HostID:            serviceInstance.HostID,
		}
		relations[idx] = relation
	}
	_, err = ps.CoreAPI.CoreService().Process().CreateProcessInstanceRelations(ctx.Kit.Ctx, ctx.Kit.Header, relations)
	if err != nil {
		blog.ErrorJSON("create service instance relations, CreateProcessInstanceRelations err: %s, relations:%s, rid: %s", err, relations, ctx.Kit.Rid)
		return nil, err
	}

	return processIDs, nil
}

func (ps *ProcServer) UpdateProcessInstancesByIDs(ctx *rest.Contexts) {
	input := metadata.UpdateProcessByIDsInput{}
	if err := ctx.DecodeInto(&input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	filter := map[string]interface{}{
		common.BKProcessIDField: map[string]interface{}{
			common.BKDBIN: input.ProcessIDs,
		},
	}
	reqParam := &metadata.QueryCondition{
		Condition: filter,
		Page:      metadata.BasePage{Limit: common.BKNoLimit},
	}
	processResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDProc, reqParam)
	if nil != err {
		ctx.RespWithError(err, common.CCErrProcGetServiceInstancesFailed, "UpdateProcessInstancesByIDs failed, reqParam: %#v, err: %+v", reqParam, err)
		return
	}

	raws := make([]map[string]interface{}, 0)
	for _, process := range processResult.Data.Info {
		for k, v := range input.UpdateData {
			process[k] = v
		}
		raws = append(raws, process)
	}

	if len(raws) == 0 {
		ctx.RespEntity([]int64{})
		return
	}

	updateInput := metadata.UpdateRawProcessInstanceInput{
		BizID: input.BizID,
		Raw:   raws,
	}

	var result []int64
	txnErr := ps.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		result, err = ps.updateProcessInstances(ctx, updateInput)
		if err != nil {
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(result)
}

func (ps *ProcServer) UpdateProcessInstances(ctx *rest.Contexts) {
	input := metadata.UpdateRawProcessInstanceInput{}
	if err := ctx.DecodeInto(&input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	if len(input.Raw) == 0 {
		ctx.RespEntity([]int64{})
		return
	}

	var result []int64
	txnErr := ps.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		result, err = ps.updateProcessInstances(ctx, input)
		if err != nil {
			return err
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(result)
}

func (ps *ProcServer) updateProcessInstances(ctx *rest.Contexts, input metadata.UpdateRawProcessInstanceInput) (
	[]int64, errors.CCErrorCoder) {

	rid := ctx.Kit.Rid

	// get process ids from the raw process data, and collect those whose update data contains bind info field
	processIDs := make([]int64, 0)
	updateBindInfoMap := make(map[int64]struct{})
	for _, pData := range input.Raw {
		processID, err := util.GetInt64ByInterface(pData[common.BKProcIDField])
		if err != nil {
			blog.ErrorJSON("update process instance, but parse id failed, data: %s, err: %s, rid: %s", pData, err, rid)
			return nil, ctx.Kit.CCError.CCError(common.CCErrCommJSONUnmarshalFailed)
		}

		if processID == 0 {
			blog.Errorf("update process instance failed, process_id invalid, rid: %s", rid)
			return nil, ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcessIDField)
		}
		processIDs = append(processIDs, processID)

		if pData[common.BKProcBindInfo] != nil {
			updateBindInfoMap[processID] = struct{}{}
		}
	}

	// get process relations to get their template ids and service instance ids
	processIDs = util.IntArrayUnique(processIDs)
	option := &metadata.ListProcessInstanceRelationOption{
		BusinessID: input.BizID,
		ProcessIDs: processIDs,
		Page:       metadata.BasePage{Limit: common.BKNoLimit},
	}
	relations, err := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(ctx.Kit.Ctx, ctx.Kit.Header, option)
	if err != nil {
		blog.ErrorJSON("search process instance relation failed, option: %s, err: %s, rid: %s", option, err, rid)
		return nil, err
	}

	// make sure all process valid
	foundProcessIDs := make(map[int64]struct{}, 0)
	for _, relation := range relations.Info {
		foundProcessIDs[relation.ProcessID] = struct{}{}
	}
	invalidProcessIDs := make([]string, 0)
	for _, processID := range processIDs {
		if _, exists := foundProcessIDs[processID]; !exists {
			invalidProcessIDs = append(invalidProcessIDs, strconv.FormatInt(processID, 10))
		}
	}
	if len(invalidProcessIDs) > 0 {
		blog.Errorf("update process instance failed, process %+v not found", invalidProcessIDs)
		msg := fmt.Sprintf("[%s: %s]", common.BKProcessIDField, strings.Join(invalidProcessIDs, ","))
		err := ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, msg)
		return nil, err
	}

	// get process to relation map and all process template ids and template related process ids
	process2ServiceInstanceMap := make(map[int64]*metadata.ProcessInstanceRelation)
	processTemplateIDs := make([]int64, 0)
	processTemplateProcessIDMap := make(map[int64][]int64)
	for i, relation := range relations.Info {
		process2ServiceInstanceMap[relation.ProcessID] = &relations.Info[i]
		if relation.ProcessTemplateID == common.ServiceTemplateIDNotSet {
			continue
		}

		processIDs, exist := processTemplateProcessIDMap[relation.ProcessTemplateID]
		if !exist {
			processTemplateIDs = append(processTemplateIDs, relation.ProcessTemplateID)
			processTemplateProcessIDMap[relation.ProcessTemplateID] = make([]int64, 0)
		}

		if _, exist := updateBindInfoMap[relation.ProcessID]; exist {
			processTemplateProcessIDMap[relation.ProcessTemplateID] = append(processIDs, relation.ProcessID)
		}
	}

	// get all process templates and those process ids that need to update bind info fields that are locked
	processTemplateMap := make(map[int64]*metadata.ProcessTemplate)
	procIDs := make([]int64, 0)
	if len(processTemplateIDs) > 0 {
		procTempOpt := &metadata.ListProcessTemplatesOption{
			BusinessID:         input.BizID,
			ProcessTemplateIDs: processTemplateIDs,
			Page:               metadata.BasePage{Limit: common.BKNoLimit},
		}
		procTemps, err := ps.CoreAPI.CoreService().Process().ListProcessTemplates(ctx.Kit.Ctx, ctx.Kit.Header, procTempOpt)
		if err != nil {
			blog.ErrorJSON("get process template failed, option: %s, err: %s, rid: %s", procTempOpt, err, rid)
			return nil, err
		}

		for _, procTemp := range procTemps.Info {
			processTemplateMap[procTemp.ID] = &procTemp

			// if bind info has locked fields, collect the process ids that need to change bind info value
			if !metadata.IsAsDefaultValue(procTemp.Property.BindInfo.AsDefaultValue) {
				continue
			}

			for _, bindInfo := range procTemp.Property.BindInfo.Value {
				if bindInfo.NeedUpdate() {
					procIDs = append(procIDs, processTemplateProcessIDMap[procTemp.ID]...)
					break
				}
			}
		}
	}

	// get bind info value for the processes that need to update bind info fields, update based on the previous data
	procMap := make(map[int64]map[string]interface{})
	if len(procIDs) > 0 {
		procCond := &metadata.QueryCondition{
			Condition: map[string]interface{}{common.BKProcessIDField: map[string]interface{}{common.BKDBIN: procIDs}},
			Fields:    []string{common.BKProcessIDField, common.BKProcBindInfo},
		}

		procRes, e := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header,
			common.BKInnerObjIDProc, procCond)
		if e != nil {
			blog.Errorf("search process by ids(%+v) failed, err: %v, rid: %s", procIDs, err, rid)
			return nil, ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
		}

		for _, proc := range procRes.Data.Info {
			processID, err := util.GetInt64ByInterface(proc[common.BKProcessIDField])
			if err != nil {
				blog.ErrorJSON("parse process id failed, data: %s, err: %s, rid: %s", proc, err, rid)
				return nil, ctx.Kit.CCError.CCError(common.CCErrCommJSONUnmarshalFailed)
			}
			procMap[processID] = proc
		}
	}

	// validate and organize the update process data
	processDataMap := make(map[int64]map[string]interface{})
	for _, process := range input.Raw {
		processID, _ := util.GetInt64ByInterface(process[common.BKProcIDField])
		relation, exist := process2ServiceInstanceMap[processID]
		if !exist {
			err := ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcessIDField)
			blog.ErrorJSON("process related relation not found, process: %s, err: %s, rid: %s", process, err, rid)
			return nil, err
		}

		if relation.ProcessTemplateID == common.ServiceTemplateIDNotSet {
			// process with no template need to validate if the instance is unique
			serviceInstanceID := relation.ServiceInstanceID
			if err := ps.validateRawInstanceUnique(ctx.Kit, input.BizID, serviceInstanceID, process); err != nil {
				blog.ErrorJSON("update process instance failed, serviceInstanceID: %s, process: %s, err: %s, rid: %s",
					serviceInstanceID, process, err, rid)
				return nil, err
			}
			delete(process, common.BKProcessIDField)
			delete(process, common.MetadataField)
			delete(process, common.LastTimeField)
			delete(process, common.CreateTimeField)
		} else {
			processTemplate, exist := processTemplateMap[relation.ProcessTemplateID]
			if !exist {
				err := ctx.Kit.CCError.CCError(common.CCErrCommNotFound)
				blog.Errorf("process related template not found, relation: %+v, err: %v, rid: %s", relation, err, rid)
				return nil, err
			}

			// for process with template, only update the unlocked fields
			editableFields := processTemplate.ExtractEditableFields()
			editableProcessData := make(map[string]interface{})
			for _, field := range editableFields {
				if value, exist := process[field]; exist {
					editableProcessData[field] = value
				}
			}
			process = editableProcessData

			// update process bind info, only update the editable fields
			if process[common.BKProcBindInfo] != nil {
				bindInfoArr, err := ps.parseBindInfo(ctx.Kit, process[common.BKProcBindInfo])
				if err != nil {
					return nil, err
				}

				procBindInfoMap := make(map[int64]metadata.ProcBindInfo, 0)
				for _, bindInfo := range bindInfoArr {
					if bindInfo.Std == nil {
						continue
					}
					procBindInfoMap[bindInfo.Std.TemplateRowID] = bindInfo
				}

				originBindInfoArr, err := ps.parseBindInfo(ctx.Kit, procMap[processID][common.BKProcBindInfo])
				if err != nil {
					return nil, err
				}

				newBindInfoArr := make([]metadata.ProcBindInfo, 0)
				templateBindInfoMap := make(map[int64]metadata.ProcPropertyBindInfoValue, 0)
				for _, row := range processTemplate.Property.BindInfo.Value {
					templateBindInfoMap[row.Std.RowID] = row
				}

				for _, bindInfo := range originBindInfoArr {
					if bindInfo.Std == nil {
						continue
					}

					inputProcBindInfo, exists := procBindInfoMap[bindInfo.Std.TemplateRowID]
					if !exists {
						newBindInfoArr = append(newBindInfoArr, bindInfo)
						continue
					}

					row, exists := templateBindInfoMap[bindInfo.Std.TemplateRowID]
					if !exist {
						newBindInfoArr = append(newBindInfoArr, inputProcBindInfo)
						continue
					}

					// for process bind info, the locked fields use the previous value
					if metadata.IsAsDefaultValue(row.Std.IP.AsDefaultValue) {
						inputProcBindInfo.Std.IP = bindInfo.Std.IP
					}
					if metadata.IsAsDefaultValue(row.Std.Port.AsDefaultValue) {
						inputProcBindInfo.Std.Port = bindInfo.Std.Port
					}
					if metadata.IsAsDefaultValue(row.Std.Protocol.AsDefaultValue) {
						inputProcBindInfo.Std.Protocol = bindInfo.Std.Protocol
					}
					if metadata.IsAsDefaultValue(row.Std.Enable.AsDefaultValue) {
						inputProcBindInfo.Std.Enable = bindInfo.Std.Enable
					}

					row.UpdateBindInfoExtraData(inputProcBindInfo)
					newBindInfoArr = append(newBindInfoArr, inputProcBindInfo)
				}

				process[common.BKProcBindInfo] = newBindInfoArr
			}

			if err := ps.validateRawProcessInstance(ctx.Kit, process); err != nil {
				blog.ErrorJSON("validate update process failed, err: %s, data: %s, rid: %s", err, process, rid)
				return nil, err
			}
		}
		processDataMap[processID] = process
	}

	var wg sync.WaitGroup
	var firstErr errors.CCErrorCoder
	pipeline := make(chan struct{}, 50)

	for processID := range processDataMap {
		pipeline <- struct{}{}
		wg.Add(1)

		go func(processID int64, processData map[string]interface{}) {
			defer func() {
				wg.Done()
				<-pipeline
			}()

			err := ps.Logic.UpdateProcessInstance(ctx.Kit, processID, processData)
			if err != nil {
				blog.ErrorJSON("update process instance failed, processID: %s, process: %s, err: %s, rid: %s",
					processID, processData, err, rid)
				if firstErr == nil {
					firstErr = err
				}
				return
			}

		}(processID, processDataMap[processID])
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return processIDs, nil
}

func (ps *ProcServer) parseBindInfo(kit *rest.Kit, bindInfo interface{}) ([]metadata.ProcBindInfo, errors.CCErrorCoder) {
	bindInfoJson, err := json.Marshal(bindInfo)
	if err != nil {
		blog.ErrorJSON("marshal bind info failed, err: %s, bind info: %s, rid: %s", err, bindInfo, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo)
	}

	bindInfoArr := make([]metadata.ProcBindInfo, 0)
	if err := json.Unmarshal(bindInfoJson, &bindInfoArr); err != nil {
		blog.Errorf("unmarshal bind info failed, err: %v, bind info: %s, rid: %s", err, string(bindInfoJson), kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo)
	}
	return bindInfoArr, nil
}

func (ps *ProcServer) CheckHostInBusiness(ctx *rest.Contexts, bizID int64, hostIDs []int64) errors.CCErrorCoder {
	hostIDHit := make(map[int64]bool)
	for _, hostID := range hostIDs {
		hostIDHit[hostID] = false
	}
	hostConfigFilter := &metadata.DistinctHostIDByTopoRelationRequest{
		ApplicationIDArr: []int64{bizID},
		HostIDArr:        hostIDs,
	}
	result, err := ps.CoreAPI.CoreService().Host().GetDistinctHostIDByTopology(ctx.Kit.Ctx, ctx.Kit.Header, hostConfigFilter)
	if err != nil {
		blog.ErrorJSON("CheckHostInBusiness failed, GetHostModuleRelation failed, filter: %s, err: %s, rid: %s", hostConfigFilter, err.Error(), ctx.Kit.Rid)
		e, ok := err.(errors.CCErrorCoder)
		if ok {
			return e
		} else {
			return ctx.Kit.CCError.CCError(common.CCErrWebGetHostFail)
		}
	}
	for _, id := range result.Data.IDArr {
		hostIDHit[id] = true
	}
	invalidHost := make([]int64, 0)
	for hostID, hit := range hostIDHit {
		if !hit {
			invalidHost = append(invalidHost, hostID)
		}
	}
	if len(invalidHost) > 0 {
		return ctx.Kit.CCError.CCErrorf(common.CCErrCoreServiceHostNotBelongBusiness, invalidHost, bizID)
	}
	return nil
}

func (ps *ProcServer) getDefaultModule(ctx *rest.Contexts, bizID int64, defaultFlag int) (*metadata.ModuleInst, errors.CCErrorCoder) {
	filter := map[string]interface{}{
		common.BKAppIDField:   bizID,
		common.BKDefaultField: defaultFlag,
	}
	return ps.getOneModule(ctx, filter)
}

func (ps *ProcServer) getModule(ctx *rest.Contexts, moduleID int64) (*metadata.ModuleInst, errors.CCErrorCoder) {
	filter := map[string]interface{}{
		common.BKModuleIDField: moduleID,
	}
	return ps.getOneModule(ctx, filter)
}

func (ps *ProcServer) getOneModule(ctx *rest.Contexts, filter map[string]interface{}) (*metadata.ModuleInst, errors.CCErrorCoder) {
	moduleFilter := &metadata.QueryCondition{
		Condition: mapstr.MapStr(filter),
	}
	modules, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, moduleFilter)
	if err != nil {
		blog.Errorf("getModule failed, filter: %+v, err: %s, rid: %s", filter, err.Error(), ctx.Kit.Rid)
		return nil, ctx.Kit.CCError.CCErrorf(common.CCErrTopoGetModuleFailed, err)
	}
	if len(modules.Data.Info) == 0 {
		blog.Errorf("getModule failed, filter: %+v, err: %+v, rid: %s", filter, "not found", ctx.Kit.Rid)
		return nil, ctx.Kit.CCError.CCErrorf(common.CCErrTopoGetModuleFailed, "not found")
	}
	if len(modules.Data.Info) > 1 {
		blog.Errorf("getModule failed, filter: %+v, err: %+v, rid: %s", filter, "get multiple", ctx.Kit.Rid)
		return nil, ctx.Kit.CCError.CCErrorf(common.CCErrTopoGetModuleFailed, "get multiple modules")
	}
	module := modules.Data.Info[0]
	moduleInst := &metadata.ModuleInst{}
	if err := module.ToStructByTag(moduleInst, "field"); err != nil {
		blog.Errorf("getModule failed, marshal json failed, filter: %+v, err: %+v, rid: %s", filter, err, ctx.Kit.Rid)
		return nil, ctx.Kit.CCError.CCErrorf(common.CCErrCommJSONUnmarshalFailed)
	}
	return moduleInst, nil
}

func (ps *ProcServer) getModules(ctx *rest.Contexts, moduleIDs []int64) ([]*metadata.ModuleInst, errors.CCErrorCoder) {
	moduleFilter := &metadata.QueryCondition{
		Condition: map[string]interface{}{
			common.BKModuleIDField: map[string]interface{}{
				common.BKDBIN: moduleIDs,
			},
		},
	}
	modules, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, moduleFilter)
	if err != nil {
		blog.Errorf("getModules failed, moduleIDs: %+v, err: %s, rid: %s", moduleIDs, err.Error(), ctx.Kit.Rid)
		return nil, ctx.Kit.CCError.CCErrorf(common.CCErrTopoGetModuleFailed, err)
	}
	moduleInsts := make([]*metadata.ModuleInst, 0)
	for _, module := range modules.Data.Info {
		moduleInst := new(metadata.ModuleInst)
		if err := module.ToStructByTag(moduleInst, "field"); err != nil {
			blog.Errorf("getModules failed, unmarshal json failed, module: %+v, err: %+v, rid: %s", module, err, ctx.Kit.Rid)
			return nil, ctx.Kit.CCError.CCErrorf(common.CCErrCommJSONUnmarshalFailed)
		}
		moduleInsts = append(moduleInsts, moduleInst)

	}
	return moduleInsts, nil
}

var (
	ipRegex = `^((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})(\.((2(5[0-5]|[0-4]\d))|[0-1]?\d{1,2})){3}$`
)

func (ps *ProcServer) validateProcessInstance(kit *rest.Kit, process *metadata.Process) errors.CCErrorCoder {
	if process.ProcessName != nil && (len(*process.ProcessName) == 0 || len(*process.ProcessName) > common.NameFieldMaxLength) {
		return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcessNameField)
	}
	if process.FuncName != nil && (len(*process.FuncName) == 0 || len(*process.ProcessName) > common.NameFieldMaxLength) {
		return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKFuncName)
	}

	// validate that process bind info must have ip and port and protocol
	for _, bindInfo := range process.BindInfo {
		if bindInfo.Std.IP == nil || len(*bindInfo.Std.IP) == 0 {
			return kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, common.BKProcBindInfo+"."+common.BKIP)
		}
		matched, err := regexp.MatchString(ipRegex, *bindInfo.Std.IP)
		if err != nil || !matched {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo+"."+common.BKIP)
		}

		if bindInfo.Std.Port == nil || len(*bindInfo.Std.Port) == 0 {
			return kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, common.BKProcBindInfo+"."+common.BKPort)
		}
		if matched := metadata.ProcessPortFormat.MatchString(*bindInfo.Std.Port); !matched {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo+"."+common.BKPort)
		}

		if bindInfo.Std.Protocol == nil || len(*bindInfo.Std.Protocol) == 0 {
			return kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, common.BKProcBindInfo+"."+common.BKProtocol)
		}
		if *bindInfo.Std.Protocol != string(metadata.ProtocolTypeTCP) && *bindInfo.Std.Protocol != string(metadata.ProtocolTypeUDP) {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo+"."+common.BKProtocol)
		}
	}

	return nil
}

func (ps *ProcServer) validateRawProcessInstance(kit *rest.Kit, process map[string]interface{}) errors.CCErrorCoder {
	processName := process[common.BKProcessNameField]
	if processName != nil {
		processNameStr := util.GetStrByInterface(processName)
		if len(processNameStr) == 0 || len(processNameStr) > common.NameFieldMaxLength {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcessNameField)
		}
	}

	processFuncName := process[common.BKFuncName]
	if processFuncName != nil {
		processFuncNameStr := util.GetStrByInterface(processFuncName)
		if len(processFuncNameStr) == 0 || len(processFuncNameStr) > common.NameFieldMaxLength {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKFuncName)
		}
	}

	// validate that process bind info must have ip and port and protocol
	bindInfoArr, err := ps.parseBindInfo(kit, process[common.BKProcBindInfo])
	if err != nil {
		return err
	}

	for _, bindInfo := range bindInfoArr {
		if bindInfo.Std.IP == nil || len(*bindInfo.Std.IP) == 0 {
			return kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, common.BKProcBindInfo+"."+common.BKIP)
		}
		matched, err := regexp.MatchString(ipRegex, *bindInfo.Std.IP)
		if err != nil || !matched {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo+"."+common.BKIP)
		}

		if bindInfo.Std.Port == nil || len(*bindInfo.Std.Port) == 0 {
			return kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, common.BKProcBindInfo+"."+common.BKPort)
		}
		if matched := metadata.ProcessPortFormat.MatchString(*bindInfo.Std.Port); !matched {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo+"."+common.BKPort)
		}

		if bindInfo.Std.Protocol == nil || len(*bindInfo.Std.Protocol) == 0 {
			return kit.CCError.CCErrorf(common.CCErrCommParamsNeedSet, common.BKProcBindInfo+"."+common.BKProtocol)
		}
		if *bindInfo.Std.Protocol != string(metadata.ProtocolTypeTCP) && *bindInfo.Std.Protocol != string(metadata.ProtocolTypeUDP) {
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcBindInfo+"."+common.BKProtocol)
		}
	}

	return nil
}

func (ps *ProcServer) validateRawInstanceUnique(kit *rest.Kit, bizID int64, serviceInstanceID int64,
	process map[string]interface{}) errors.CCErrorCoder {

	if err := ps.validateRawProcessInstance(kit, process); err != nil {
		blog.ErrorJSON("validate process instance failed, err: %s, process: %s, rid: %s", err, process, kit.Rid)
		return err
	}

	processName := process[common.BKProcessNameField]
	if processName == nil {
		return nil
	}
	processName = util.GetStrByInterface(processName)

	// find process under service instance
	relationOption := &metadata.ListProcessInstanceRelationOption{
		BusinessID:         bizID,
		ServiceInstanceIDs: []int64{serviceInstanceID},
		ProcessTemplateID:  common.ServiceTemplateIDNotSet,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	relations, err := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(kit.Ctx, kit.Header, relationOption)
	if err != nil {
		blog.Errorf("get relation under service instance %d failed, err: %v, rid: %s", serviceInstanceID, err, kit.Rid)
		return kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	existProcessIDs := make([]int64, 0)
	processID, _ := util.GetInt64ByInterface(process[common.BKProcIDField])
	for _, relation := range relations.Info {
		// exclude the process itself
		if relation.ProcessID != processID {
			existProcessIDs = append(existProcessIDs, relation.ProcessID)
		}
	}
	if len(existProcessIDs) == 0 {
		return nil
	}

	filter := map[string]interface{}{
		common.BKProcessIDField: map[string]interface{}{
			common.BKDBIN: existProcessIDs,
		},
		common.BKProcessNameField: processName,
	}

	filterCond := &metadata.QueryCondition{
		Condition: mapstr.MapStr(filter),
		Fields:    []string{common.BKProcessIDField},
	}

	listResult, e := ps.CoreAPI.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, common.BKProcessObjectName, filterCond)
	if e != nil {
		blog.ErrorJSON("validateManyRawInstanceUnique failed, search process with bk_process_name failed, filterCond: %s, err: %s, rid: %s",
			filterCond, e, kit.Rid)
		return kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	if len(listResult.Data.Info) > 0 {
		blog.ErrorJSON("bk_process_name duplicated under service instance, serviceInstance: %s, filterCond: %s, "+
			"duplicate processes: %s, rid: %s", serviceInstanceID, filterCond, listResult.Data.Info, kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrCoreServiceProcessNameDuplicated, processName)
	}

	return nil
}

func (ps *ProcServer) validateManyRawInstanceUnique(kit *rest.Kit, serviceInstance *metadata.ServiceInstance,
	processDatas []map[string]interface{}) errors.CCErrorCoder {

	processNamesMap := make(map[string]bool)
	originalProcessIDMap := make(map[int64]bool)

	for _, processData := range processDatas {
		if err := ps.validateRawProcessInstance(kit, processData); err != nil {
			blog.ErrorJSON("valid process(%s) failed, err: %s, rid: %s", processData, err, kit.Rid)
			return err
		}

		processName := processData[common.BKProcessNameField]
		if processName == nil {
			continue
		}
		processNameStr := util.GetStrByInterface(processName)

		// check if original process data contains duplicate name
		if processNamesMap[processNameStr] {
			return kit.CCError.CCErrorf(common.CCErrCoreServiceProcessNameDuplicated, processNameStr)
		}
		processNamesMap[processNameStr] = true

		if processData[common.BKProcessIDField] == nil {
			continue
		}

		processID, err := util.GetInt64ByInterface(processData[common.BKProcessIDField])
		if err != nil {
			blog.ErrorJSON("parse process id failed, process: %s, err: %s, rid: %s", processData, err, kit.Rid)
			return kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKProcessIDField)
		}

		if processID != 0 {
			originalProcessIDMap[processID] = true
		}
	}

	// find process under service instance
	bizID := serviceInstance.BizID
	relationOption := &metadata.ListProcessInstanceRelationOption{
		BusinessID:         bizID,
		ServiceInstanceIDs: []int64{serviceInstance.ID},
		ProcessTemplateID:  common.ServiceTemplateIDNotSet,
		HostID:             serviceInstance.HostID,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	relations, err := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(kit.Ctx, kit.Header, relationOption)
	if err != nil {
		blog.Errorf("validate many raw instance unique failed, get relation under service instance failed, "+
			"serviceInstanceID: %d, err: %v, rid: %s", serviceInstance.ID, err, kit.Rid)
		return kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	existProcessIDs := make([]int64, 0)
	for _, relation := range relations.Info {
		// exclude the processes themselves
		if originalProcessIDMap[relation.ProcessID] != true {
			existProcessIDs = append(existProcessIDs, relation.ProcessID)
		}
	}
	if len(existProcessIDs) == 0 {
		return nil
	}

	filter := map[string]interface{}{
		common.BKProcessIDField: map[string]interface{}{
			common.BKDBIN: existProcessIDs,
		},
	}

	filterCond := &metadata.QueryCondition{
		Condition: mapstr.MapStr(filter),
		Fields:    []string{common.BKProcessNameField},
	}

	listResult, e := ps.CoreAPI.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, common.BKProcessObjectName, filterCond)
	if e != nil {
		blog.ErrorJSON("validate many raw instance unique failed, search process with bk_process_name failed, "+
			"filterCond: %s, err: %s, rid: %s", filterCond, e, kit.Rid)
		return kit.CCError.CCError(common.CCErrCommDBSelectFailed)
	}

	for _, proc := range listResult.Data.Info {
		// check process name unique
		if procName, _ := proc.String(common.BKProcessNameField); processNamesMap[procName] {
			blog.ErrorJSON("validate many raw instance unique failed, process name duplicated under service instance, "+
				"serviceInstanceID: %s, filterCond: %s, err: %s, rid: %s", serviceInstance.ID, filterCond, err, kit.Rid)
			return kit.CCError.CCErrorf(common.CCErrCoreServiceProcessNameDuplicated, procName)
		}
	}

	return nil
}

func (ps *ProcServer) DeleteProcessInstance(ctx *rest.Contexts) {
	input := new(metadata.DeleteProcessInstanceInServiceInstanceInput)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	listOption := &metadata.ListProcessInstanceRelationOption{
		BusinessID: input.BizID,
		ProcessIDs: input.ProcessInstanceIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	relations, err := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(ctx.Kit.Ctx, ctx.Kit.Header, listOption)
	if err != nil {
		blog.Errorf("DeleteProcessInstance failed, ListProcessInstanceRelation failed, option: %+v, err: %+v, rid: %s", listOption, err, ctx.Kit.Rid)
		ctx.RespWithError(err, common.CCErrProcDeleteProcessFailed, "delete process instance: %+v, but list instance relation failed.", input.ProcessInstanceIDs)
		return
	}
	templateProcessIDs := make([]string, 0)
	for _, relation := range relations.Info {
		if relation.ProcessTemplateID != common.ServiceTemplateIDNotSet {
			templateProcessIDs = append(templateProcessIDs, strconv.FormatInt(relation.ProcessID, 10))
		}
	}
	if len(templateProcessIDs) > 0 {
		invalidProcesses := strings.Join(templateProcessIDs, ",")
		blog.Errorf("DeleteProcessInstance failed, some process:%s initialized by template, rid: %s", invalidProcesses, ctx.Kit.Rid)
		err := ctx.Kit.CCError.CCErrorf(common.CCErrCoreServiceShouldNotRemoveProcessCreateByTemplate, invalidProcesses)
		ctx.RespWithError(err, common.CCErrProcDeleteProcessFailed, "delete process instance: %v, but delete instance relation failed.", input.ProcessInstanceIDs)
		return
	}

	txnErr := ps.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		// delete process relation at the same time.
		deleteOption := metadata.DeleteProcessInstanceRelationOption{}
		deleteOption.ProcessIDs = input.ProcessInstanceIDs
		err = ps.CoreAPI.CoreService().Process().DeleteProcessInstanceRelation(ctx.Kit.Ctx, ctx.Kit.Header, deleteOption)
		if err != nil {
			blog.Errorf("delete process instance: %v, but delete instance relation failed.", input.ProcessInstanceIDs)
			return ctx.Kit.CCError.CCError(common.CCErrProcDeleteProcessFailed)
		}

		if err := ps.Logic.DeleteProcessInstanceBatch(ctx.Kit, input.ProcessInstanceIDs); err != nil {
			blog.Errorf("delete process instance:%v failed, err: %v", input.ProcessInstanceIDs, err)
			return ctx.Kit.CCError.CCError(common.CCErrProcDeleteProcessFailed)
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}
	ctx.RespEntity(nil)
}

func (ps *ProcServer) ListProcessInstances(ctx *rest.Contexts) {
	input := new(metadata.ListProcessInstancesOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	processInstanceList, err := ps.Logic.ListProcessInstances(ctx.Kit, input.BizID, input.ServiceInstanceID, nil)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntity(processInstanceList)
}

// ListProcessInstancesNameIDsInModule get the process id list with its name in a module
func (ps *ProcServer) ListProcessInstancesNameIDsInModule(ctx *rest.Contexts) {
	input := new(metadata.ListProcessInstancesNameIDsOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	option := &metadata.DistinctFieldOption{
		TableName: common.BKTableNameServiceInstance,
		Field:     common.BKFieldID,
		Filter: map[string]interface{}{
			common.BKAppIDField:    input.BizID,
			common.BKModuleIDField: input.ModuleID,
		},
	}
	sIDs, err := ps.CoreAPI.CoreService().Common().GetDistinctField(ctx.Kit.Ctx, ctx.Kit.Header, option)
	if err != nil {
		blog.Errorf("GetDistinctField failed, err:%s, option:%#v, rid:%s", err, *option, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}
	if len(sIDs) == 0 {
		ctx.RespEntityWithCount(0, []map[string][]int64{})
		return
	}

	serviceInstanceIDs := make([]int64, len(sIDs))
	for idx, sID := range sIDs {
		if ID, err := strconv.ParseInt(fmt.Sprintf("%v", sID), 10, 64); err == nil {
			serviceInstanceIDs[idx] = ID
		}
	}
	listRelationOption := &metadata.ListProcessInstanceRelationOption{
		BusinessID:         input.BizID,
		ServiceInstanceIDs: serviceInstanceIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	relations, err := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(ctx.Kit.Ctx, ctx.Kit.Header, listRelationOption)
	if err != nil {
		ctx.RespWithError(err, common.CCErrProcGetProcessInstanceRelationFailed, "ListProcessInstancesNameIDsInModule failed, list option: %+v, err: %+v", listRelationOption, err)
		return
	}

	processIDs := make([]int64, 0)
	for _, relation := range relations.Info {
		processIDs = append(processIDs, relation.ProcessID)
	}

	filter := map[string]interface{}{
		common.BKProcessIDField: map[string]interface{}{
			common.BKDBIN: processIDs,
		},
	}
	if input.ProcessName != "" {
		filter[common.BKProcessNameField] = map[string]interface{}{common.BKDBLIKE: input.ProcessName, common.BKDBOPTIONS: "i"}
	}
	sort := common.BKProcessNameField
	if input.Page.Sort == "-"+common.BKProcessNameField {
		sort = input.Page.Sort
	}
	reqParam := &metadata.QueryCondition{
		Condition: filter,
		Fields:    []string{common.BKProcessIDField, common.BKProcessNameField},
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
			Sort:  sort,
		},
	}
	processResult, ccErr := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDProc, reqParam)
	if nil != ccErr {
		ctx.RespWithError(err, common.CCErrProcGetServiceInstancesFailed, "ListProcessInstancesNameIDsInModule failed, reqParam: %#v, err: %+v", reqParam, ccErr)
		return
	}

	processNameIDs := make(map[string][]int64)
	sortedProcessNames := make([]string, 0)

	for _, process := range processResult.Data.Info {
		processID, err := process.Int64(common.BKProcessIDField)
		if err != nil {
			ctx.RespWithError(err, common.CCErrCommParseDataFailed, "ListProcessInstancesNameIDsInModule failed, process: %#v, err: %+v", process, err)
			return
		}
		processName, err := process.String(common.BKProcessNameField)
		if err != nil {
			ctx.RespWithError(err, common.CCErrCommParseDataFailed, "ListProcessInstancesNameIDsInModule failed, process: %#v, err: %+v", process, err)
			return
		}
		if _, ok := processNameIDs[processName]; !ok {
			processNameIDs[processName] = make([]int64, 0)
			sortedProcessNames = append(sortedProcessNames, processName)
		}
		processNameIDs[processName] = append(processNameIDs[processName], processID)
	}

	startIndex := input.Page.Start
	if startIndex >= len(sortedProcessNames) {
		ctx.RespEntityWithCount(int64(len(sortedProcessNames)), []map[string][]int64{})
		return
	}

	endindex := startIndex + input.Page.Limit
	if endindex > len(sortedProcessNames) {
		endindex = len(sortedProcessNames)
	}

	ret := make([]metadata.ProcessInstanceNameIDs, endindex-startIndex)
	for idx, name := range sortedProcessNames[startIndex:endindex] {
		ret[idx] = metadata.ProcessInstanceNameIDs{
			ProcessName: name,
			ProcessIDs:  processNameIDs[name],
		}
	}

	ctx.RespEntityWithCount(int64(len(sortedProcessNames)), ret)
}

// ListProcessRelatedInfo list process related info according to condition
func (ps *ProcServer) ListProcessRelatedInfo(ctx *rest.Contexts) {

	bizID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKAppIDField), 10, 64)
	if err != nil {
		blog.Errorf("ListProcessRelatedInfo failed, parse bk_biz_id error, err: %s, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, "bk_biz_id"))
		return
	}

	input := new(metadata.ListProcessRelatedInfoOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	// get moduleIDs
	moduleIDs := input.Module.ModuleIDs
	if len(input.Set.SetIDs) > 0 {
		filter := map[string]interface{}{
			common.BKAppIDField: bizID,
			common.BKSetIDField: map[string]interface{}{
				common.BKDBIN: input.Set.SetIDs,
			},
		}
		if len(input.Module.ModuleIDs) > 0 {
			filter[common.BKModuleIDField] = map[string]interface{}{
				common.BKDBIN: input.Module.ModuleIDs,
			}
		}

		param := &metadata.QueryCondition{
			Condition: filter,
			Fields:    []string{common.BKModuleIDField},
			Page: metadata.BasePage{
				Limit: common.BKNoLimit,
			},
		}

		moduleResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, param)
		if nil != err {
			blog.Errorf("ListProcessRelatedInfo failed, coreservice http ReadInstance fail, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
			ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
			return
		}
		if !moduleResult.Result {
			blog.Errorf("ListProcessRelatedInfo failed, param: %v, err: %v, rid:%s", param, err, ctx.Kit.Rid)
			ctx.RespAutoError(moduleResult.CCError())
		}

		if len(moduleResult.Data.Info) == 0 {
			ctx.RespEntityWithCount(0, []interface{}{})
			return
		}

		mIDs := make([]int64, len(moduleResult.Data.Info))
		for idx, info := range moduleResult.Data.Info {
			mID, _ := info.Int64(common.BKModuleIDField)
			mIDs[idx] = mID
		}

		moduleIDs = mIDs
	}

	// get serviceIntanceIDs
	serviceIntanceIDs := input.ServiceInstance.IDs
	if len(input.ServiceInstance.IDs) > 0 || len(moduleIDs) > 0 {
		filter := map[string]interface{}{
			common.BKAppIDField: bizID,
		}

		if len(input.ServiceInstance.IDs) > 0 {
			filter[common.BKFieldID] = map[string]interface{}{
				common.BKDBIN: input.ServiceInstance.IDs,
			}
		}

		if len(moduleIDs) > 0 {
			filter[common.BKModuleIDField] = map[string]interface{}{
				common.BKDBIN: moduleIDs,
			}
		}

		option := &metadata.DistinctFieldOption{
			TableName: common.BKTableNameServiceInstance,
			Field:     common.BKFieldID,
			Filter:    filter,
		}

		sIDs, err := ps.CoreAPI.CoreService().Common().GetDistinctField(ctx.Kit.Ctx, ctx.Kit.Header, option)
		if err != nil {
			blog.Errorf("GetDistinctField failed, err:%s, option:%#v, rid:%s", err, *option, ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}

		if len(sIDs) == 0 {
			ctx.RespEntityWithCount(0, []interface{}{})
			return
		}

		srvInstIDs := make([]int64, len(sIDs))
		for idx, sID := range sIDs {
			if ID, err := strconv.ParseInt(fmt.Sprintf("%v", sID), 10, 64); err == nil {
				srvInstIDs[idx] = ID
			}
		}

		serviceIntanceIDs = srvInstIDs
	}

	// get processIDs
	var processIDs []int64
	if len(serviceIntanceIDs) > 0 {
		filter := map[string]interface{}{
			common.BKAppIDField: bizID,
			common.BKServiceInstanceIDField: map[string]interface{}{
				common.BKDBIN: serviceIntanceIDs,
			},
		}

		option := &metadata.DistinctFieldOption{
			TableName: common.BKTableNameProcessInstanceRelation,
			Field:     common.BKProcessIDField,
			Filter:    filter,
		}

		pIDs, err := ps.CoreAPI.CoreService().Common().GetDistinctField(ctx.Kit.Ctx, ctx.Kit.Header, option)
		if err != nil {
			blog.Errorf("GetDistinctField failed, err:%s, option:%#v, rid:%s", err, *option, ctx.Kit.Rid)
			ctx.RespAutoError(err)
			return
		}

		if len(pIDs) == 0 {
			ctx.RespEntityWithCount(0, []interface{}{})
			return
		}

		procIDs := make([]int64, len(pIDs))
		for idx, pID := range pIDs {
			if ID, err := strconv.ParseInt(fmt.Sprintf("%v", pID), 10, 64); err == nil {
				procIDs[idx] = ID
			}
		}

		processIDs = procIDs
	}

	// process detail
	filter := map[string]interface{}{
		common.BKAppIDField: bizID,
	}

	if len(processIDs) > 0 {
		filter[common.BKProcessIDField] = map[string]interface{}{
			common.BKDBIN: processIDs,
		}
	}

	propertyFilter := make(map[string]interface{})
	if input.ProcessPropertyFilter != nil {
		mgoFilter, key, err := input.ProcessPropertyFilter.ToMgo()
		if err != nil {
			blog.ErrorJSON("ListProcessRelatedInfo failed, ToMgo err:%s, ProcessPropertyFilter:%s, rid:%s", err, input.ProcessPropertyFilter, ctx.Kit.Rid)
			ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, err.Error()+fmt.Sprintf(", host_property_filter.%s", key)))
			return
		}
		if len(mgoFilter) > 0 {
			propertyFilter = mgoFilter
		}
	}

	finalFilter := make(map[string]interface{})
	if len(propertyFilter) > 0 {
		finalFilter[common.BKDBAND] = []map[string]interface{}{filter, propertyFilter}
	} else {
		finalFilter = filter
	}

	fields := []string{}
	if len(input.Fields) > 0 {
		fields = input.Fields
		fields = append(fields, common.BKProcessIDField)
		fields = append(fields, common.BKProcessNameField)
		fields = append(fields, common.BKFuncIDField)
	}

	sort := input.Page.Sort
	if sort == "" {
		sort = common.BKProcessIDField
	}
	reqParam := &metadata.QueryCondition{
		Fields: fields,
		Page: metadata.BasePage{
			Sort:  sort,
			Limit: input.Page.Limit,
			Start: input.Page.Start,
		},
		Condition: finalFilter,
	}

	processResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDProc, reqParam)
	if nil != err {
		blog.Errorf("ListProcessRelatedInfo failed, coreservice http ReadInstance fail, reqParam: %v, err: %v, rid:%s", *reqParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if !processResult.Result {
		blog.Errorf("ListProcessRelatedInfo failed, reqParam: %v, err: %v, rid:%s", *reqParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(processResult.CCError())
	}

	if len(processResult.Data.Info) == 0 {
		ctx.RespEntityWithCount(0, []interface{}{})
		return
	}

	processIDsNeed := make([]int64, len(processResult.Data.Info))
	processDetailMap := map[int64]interface{}{}
	for idx, process := range processResult.Data.Info {
		processID, _ := process.Int64(common.BKProcessIDField)
		processIDsNeed[idx] = processID
		processDetailMap[processID] = process
	}

	ps.listProcessRelatedInfo(ctx, bizID, processIDsNeed, processDetailMap, int64(processResult.Data.Count))
}

// listProcessRelatedInfo list process related info according to process info
func (ps *ProcServer) listProcessRelatedInfo(ctx *rest.Contexts, bizID int64, processIDs []int64,
	processDetailMap map[int64]interface{}, totalCnt int64) {

	// objID array
	srvinstArr := make([]int64, 0)
	hostArr := make([]int64, 0)
	moduleArr := make([]int64, 0)
	setArr := make([]int64, 0)

	// procID => objID map
	procSrvinstMap := make(map[int64]int64)
	procTemplateMap := make(map[int64]int64)
	procHostMap := make(map[int64]int64)
	srvinstModuleMap := make(map[int64]int64)
	moduleSetMap := make(map[int64]int64)

	// objID => objDetail map
	srvinstDetailMap := make(map[int64]metadata.ServiceInstanceDetailOfP)
	hostDetailMap := make(map[int64]metadata.HostDetailOfP)
	moduleDetailMap := make(map[int64]metadata.ModuleDetailOfP)
	setDetailMap := make(map[int64]metadata.SetDetailOfP)

	// get ID of serviceInstance, host, processTemplate and their process relation map
	listRelationOption := &metadata.ListProcessInstanceRelationOption{
		BusinessID: bizID,
		ProcessIDs: processIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	relations, ccErr := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(ctx.Kit.Ctx, ctx.Kit.Header, listRelationOption)
	if ccErr != nil {
		ctx.RespWithError(ccErr, ccErr.GetCode(), "ListProcessInstanceRelation failed, option: %+v, err: %+v", listRelationOption, ccErr)
		return
	}

	for _, relation := range relations.Info {
		srvinstArr = append(srvinstArr, relation.ServiceInstanceID)
		hostArr = append(hostArr, relation.HostID)
		procSrvinstMap[relation.ProcessID] = relation.ServiceInstanceID
		procTemplateMap[relation.ProcessID] = relation.ProcessTemplateID
		procHostMap[relation.ProcessID] = relation.HostID
		procTemplateMap[relation.ProcessID] = relation.ProcessTemplateID
	}
	srvinstArr = util.IntArrayUnique(srvinstArr)

	// service instance detail
	instOpt := &metadata.ListServiceInstanceOption{
		BusinessID:         bizID,
		ServiceInstanceIDs: srvinstArr,
	}
	instances, ccErr := ps.CoreAPI.CoreService().Process().ListServiceInstance(ctx.Kit.Ctx, ctx.Kit.Header, instOpt)
	if ccErr != nil {
		ctx.RespWithError(ccErr, ccErr.GetCode(), "ListServiceInstance failed, instOpt:%#v, err: %v", instOpt, ccErr)
		return
	}

	for _, inst := range instances.Info {
		srvinstDetailMap[inst.ID] = metadata.ServiceInstanceDetailOfP{
			ID:   inst.ID,
			Name: inst.Name,
		}
		srvinstModuleMap[inst.ID] = inst.ModuleID
		moduleArr = append(moduleArr, inst.ModuleID)
	}
	moduleArr = util.IntArrayUnique(moduleArr)

	// host detail
	hostParam := &metadata.QueryCondition{
		Fields: []string{common.BKHostIDField, common.BKCloudIDField, common.BKHostInnerIPField},
		Condition: map[string]interface{}{common.BKHostIDField: map[string]interface{}{
			common.BKDBIN: hostArr,
		},
		},
	}

	hostResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDHost, hostParam)
	if nil != err {
		blog.Errorf("ListProcessRelatedInfo failed, coreservice http ReadInstance fail, param: %v, err: %v, rid:%s", *hostParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if !hostResult.Result {
		blog.Errorf("ListProcessRelatedInfo failed, param: %v, err: %v, rid:%s", *hostParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(hostResult.CCError())
	}

	for _, host := range hostResult.Data.Info {
		hostID, _ := host.Int64(common.BKHostIDField)
		cloudID, _ := host.Int64(common.BKCloudIDField)
		innerIP, _ := host.String(common.BKHostInnerIPField)
		hostDetailMap[hostID] = metadata.HostDetailOfP{
			HostID:  hostID,
			CloudID: cloudID,
			InnerIP: innerIP,
		}
	}

	// module detail
	moduleParam := &metadata.QueryCondition{
		Fields: []string{common.BKModuleIDField, common.BKModuleNameField, common.BKSetIDField},
		Condition: map[string]interface{}{
			common.BKAppIDField: bizID,
			common.BKModuleIDField: map[string]interface{}{
				common.BKDBIN: moduleArr,
			},
		},
	}

	moduleResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDModule, moduleParam)
	if nil != err {
		blog.Errorf("ListProcessRelatedInfo failed, coreservice http ReadInstance fail, param: %v, err: %v, rid:%s", *moduleParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if !moduleResult.Result {
		blog.Errorf("ListProcessRelatedInfo failed, param: %v, err: %v, rid:%s", *moduleParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(moduleResult.CCError())
	}

	for _, module := range moduleResult.Data.Info {
		moduleID, _ := module.Int64(common.BKModuleIDField)
		moduleName, _ := module.String(common.BKModuleNameField)
		moduleDetailMap[moduleID] = metadata.ModuleDetailOfP{
			ModuleID:   moduleID,
			ModuleName: moduleName,
		}

		setID, _ := module.Int64(common.BKSetIDField)
		moduleSetMap[moduleID] = setID
		setArr = append(setArr, setID)

	}
	setArr = util.IntArrayUnique(setArr)

	// set detail
	setParam := &metadata.QueryCondition{
		Fields: []string{common.BKSetIDField, common.BKSetNameField, common.BKSetEnvField},
		Condition: map[string]interface{}{
			common.BKAppIDField: bizID,
			common.BKSetIDField: map[string]interface{}{
				common.BKDBIN: setArr,
			},
		},
	}

	setResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDSet, setParam)
	if nil != err {
		blog.Errorf("ListProcessRelatedInfo failed, coreservice http ReadInstance fail, param: %v, err: %v, rid:%s", *setParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if !setResult.Result {
		blog.Errorf("ListProcessRelatedInfo failed, param: %v, err: %v, rid:%s", *setParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(setResult.CCError())
	}

	for _, set := range setResult.Data.Info {
		setID, _ := set.Int64(common.BKSetIDField)
		setName, _ := set.String(common.BKSetNameField)
		setEnv, _ := set.String(common.BKSetEnvField)
		setDetailMap[setID] = metadata.SetDetailOfP{
			SetID:   setID,
			SetName: setName,
			SetEnv:  setEnv,
		}
	}

	// construct the final result
	ret := make([]metadata.ListProcessRelatedInfoResult, len(processIDs))

	for idx, processID := range processIDs {

		srvinstID := procSrvinstMap[processID]
		moduleID := srvinstModuleMap[srvinstID]
		setID := moduleSetMap[moduleID]

		hostDetail := hostDetailMap[procHostMap[processID]]
		srvinstDetail := srvinstDetailMap[srvinstID]
		moduleDetail := moduleDetailMap[moduleID]
		setDetail := setDetailMap[setID]

		info := metadata.ListProcessRelatedInfoResult{
			Set:             setDetail,
			Module:          moduleDetail,
			Host:            hostDetail,
			ServiceInstance: srvinstDetail,
			ProcessTemplate: metadata.ProcessTemplateDetailOfP{
				ID: procTemplateMap[processID],
			},
			Process: processDetailMap[processID],
		}
		ret[idx] = info
	}

	ctx.RespEntityWithCount(totalCnt, ret)
}

// ListProcessInstancesDetailsByIDs get process instances details and relation by their ids
func (ps *ProcServer) ListProcessInstancesDetailsByIDs(ctx *rest.Contexts) {
	input := new(metadata.ListProcessInstancesDetailsByIDsOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	filter := map[string]interface{}{
		common.BKProcessIDField: map[string]interface{}{
			common.BKDBIN: input.ProcessIDs,
		},
	}
	reqParam := &metadata.QueryCondition{
		Condition: filter,
		Page:      input.Page,
	}
	processResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDProc, reqParam)
	if nil != err {
		ctx.RespWithError(err, common.CCErrProcGetServiceInstancesFailed, "ListProcessInstancesDetailsByIDs failed, reqParam: %#v, err: %+v", reqParam, err)
		return
	}

	processIDPropertyMap := map[int64]mapstr.MapStr{}
	sortedprocessIDs := make([]int64, 0)
	for _, process := range processResult.Data.Info {
		processID, err := process.Int64(common.BKProcessIDField)
		if err != nil {
			ctx.RespWithError(err, common.CCErrCommParseDataFailed, "ListProcessInstancesDetailsByIDs failed, process: %#v, err: %+v", process, err)
			return
		}
		processIDPropertyMap[processID] = process
		sortedprocessIDs = append(sortedprocessIDs, processID)
	}

	listRelationOption := &metadata.ListProcessInstanceRelationOption{
		BusinessID: input.BizID,
		ProcessIDs: sortedprocessIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	relations, err := ps.CoreAPI.CoreService().Process().ListProcessInstanceRelation(ctx.Kit.Ctx, ctx.Kit.Header, listRelationOption)
	if err != nil {
		ctx.RespWithError(err, common.CCErrProcGetProcessInstanceRelationFailed, "ListProcessInstancesDetailsByIDs failed, list option: %+v, err: %+v", listRelationOption, err)
		return
	}

	processIDRelationMap := make(map[int64]metadata.ProcessInstanceRelation)
	serviceInstanceIDs := make([]int64, 0)
	for _, relation := range relations.Info {
		processIDRelationMap[relation.ProcessID] = relation
		serviceInstanceIDs = append(serviceInstanceIDs, relation.ServiceInstanceID)
	}

	option := &metadata.ListServiceInstanceOption{
		BusinessID:         input.BizID,
		ServiceInstanceIDs: serviceInstanceIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	serviceInstanceResult, err := ps.CoreAPI.CoreService().Process().ListServiceInstance(ctx.Kit.Ctx, ctx.Kit.Header, option)
	if err != nil {
		ctx.RespWithError(err, common.CCErrProcGetServiceInstancesFailed, "ListProcessInstancesDetailsByIDs failed, option: %#v, err: %v", option, err)
		return
	}
	serviceInstanceIDNames := make(map[int64]string)
	for _, instance := range serviceInstanceResult.Info {
		serviceInstanceIDNames[instance.ID] = instance.Name
	}

	processInstanceList := make([]metadata.ProcessInstanceDetailByID, 0)
	for _, id := range sortedprocessIDs {
		processDetail := metadata.ProcessInstanceDetailByID{
			ProcessID: id,
			Property:  processIDPropertyMap[id],
		}
		relation, exist := processIDRelationMap[id]
		if exist {
			processDetail.Relation = relation
			processDetail.ServiceInstanceName = serviceInstanceIDNames[relation.ServiceInstanceID]

		}
		processInstanceList = append(processInstanceList, processDetail)
	}

	ctx.RespEntityWithCount(int64(processResult.Data.Count), processInstanceList)
}

// ListProcessInstancesDetails get process instances details by their ids
func (ps *ProcServer) ListProcessInstancesDetails(ctx *rest.Contexts) {

	bizID, err := strconv.ParseInt(ctx.Request.PathParameter(common.BKAppIDField), 10, 64)
	if err != nil {
		blog.Errorf("ListProcessRelatedInfo failed, parse bk_biz_id error, err: %s, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, "bk_biz_id"))
		return
	}

	input := new(metadata.ListProcessInstancesDetailsOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	filter := map[string]interface{}{
		common.BKAppIDField: bizID,
		common.BKProcessIDField: map[string]interface{}{
			common.BKDBIN: input.ProcessIDs,
		},
	}

	reqParam := &metadata.QueryCondition{
		Condition: filter,
		Fields:    input.Fields,
	}

	processResult, err := ps.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDProc, reqParam)
	if nil != err {
		blog.Errorf("ListProcessInstancesDetails failed, coreservice http ReadInstance fail, reqParam: %v, err: %v, rid:%s", *reqParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if !processResult.Result {
		blog.Errorf("ListProcessInstancesDetails failed, reqParam: %v, err: %v, rid:%s", *reqParam, err, ctx.Kit.Rid)
		ctx.RespAutoError(processResult.CCError())
	}

	ctx.RespEntity(processResult.Data.Info)
}

var UnbindServiceTemplateOnModuleEnable = true

func (ps *ProcServer) RemoveTemplateBindingOnModule(ctx *rest.Contexts) {
	if UnbindServiceTemplateOnModuleEnable {
		ctx.RespErrorCodeOnly(common.CCErrProcUnbindModuleServiceTemplateDisabled, "unbind service template from module disabled")
		return
	}

	input := new(metadata.RemoveTemplateBindingOnModuleOption)
	if err := ctx.DecodeInto(input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	module, err := ps.getModule(ctx, input.ModuleID)
	if err != nil {
		ctx.RespWithError(err, common.CCErrTopoGetModuleFailed, "create service instance failed, get module failed, moduleID: %d, err: %v", input.ModuleID, err)
		return
	}
	if module.BizID != input.BizID {
		err := ctx.Kit.CCError.CCError(common.CCErrCommNotFound)
		ctx.RespWithError(err, common.CCErrCommNotFound, "create service instance failed, get module failed, moduleID: %d, err: %v", input.ModuleID, err)
		return
	}

	var response *metadata.RemoveTemplateBoundOnModuleResult
	txnErr := ps.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		response, err = ps.CoreAPI.CoreService().Process().RemoveTemplateBindingOnModule(ctx.Kit.Ctx, ctx.Kit.Header, input.ModuleID)
		if err != nil {
			blog.Errorf("remove template binding on module failed, parse business id failed, err: %+v", err)
			return ctx.Kit.CCError.CCError(common.CCErrProcRemoveTemplateBindingOnModule)
		}
		return nil
	})

	if txnErr != nil {
		blog.Errorf("RemoveTemplateBindingOnModule failed, err: %v, rid: %s", txnErr, ctx.Kit.Rid)
		return
	}
	ctx.RespEntity(response)
}

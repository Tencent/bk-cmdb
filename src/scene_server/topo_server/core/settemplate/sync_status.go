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

package settemplate

import (
	"time"

	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/lock"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/mapstruct"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
	"configcenter/src/storage/driver/redis"
)

func (st *setTemplate) GetOneSet(kit *rest.Kit, setID int64) (metadata.SetInst, errors.CCErrorCoder) {
	set := metadata.SetInst{}

	filter := map[string]interface{}{
		common.BKSetIDField: setID,
	}
	qc := &metadata.QueryCondition{
		Condition: filter,
	}
	instResult, err := st.client.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, common.BKInnerObjIDSet, qc)
	if err != nil {
		blog.ErrorJSON("GetOneSet failed, db select failed, filter: %s, err: %s, rid: %s", filter, err.Error(), kit.Rid)
		return set, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}
	if ccErr := instResult.CCError(); ccErr != nil {
		blog.ErrorJSON("GetOneSet failed, read instance failed, filter: %s, instResult: %s, rid: %s", filter, instResult, kit.Rid)
		return set, ccErr
	}
	if len(instResult.Data.Info) == 0 {
		blog.ErrorJSON("GetOneSet failed, not found, filter: %s, instResult: %s, rid: %s", filter, instResult, kit.Rid)
		return set, kit.CCError.CCError(common.CCErrCommNotFound)
	}
	if len(instResult.Data.Info) > 1 {
		blog.ErrorJSON("GetOneSet failed, got multiple, filter: %s, instResult: %s, rid: %s", filter, instResult, kit.Rid)
		return set, kit.CCError.CCError(common.CCErrCommGetMultipleObject)
	}
	if err := mapstruct.Decode2StructWithHook(instResult.Data.Info[0], &set); err != nil {
		blog.ErrorJSON("GetOneSet failed, unmarshal set failed, instResult: %s, err: %s, rid: %s", instResult, err.Error(), kit.Rid)
		return set, kit.CCError.CCError(common.CCErrCommJSONUnmarshalFailed)
	}
	return set, nil
}

func (st *setTemplate) GetSets(kit *rest.Kit, setTemplateID int64, setIDs []int64) ([]metadata.SetInst, errors.CCErrorCoder) {
	filter := &metadata.QueryCondition{}
	filter.Condition = mapstr.MapStr{
		common.BKSetIDField:         map[string]interface{}{common.BKDBIN: setIDs},
		common.BKSetTemplateIDField: setTemplateID,
	}

	instResult := new(metadata.ResponseSetInstance)
	err := st.client.CoreService().Instance().ReadInstanceStruct(kit.Ctx, kit.Header, common.BKInnerObjIDSet, filter, instResult)
	if err != nil {
		blog.Errorf("GetSets failed, db select failed, filter: %s, err: %s, rid: %s", filter, err.Error(), kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}

	if ccErr := instResult.CCError(); ccErr != nil {
		blog.Errorf("GetSets failed, read instance failed, filter: %s, instResult: %s, err: %s, rid: %s",
			filter, instResult, ccErr.Error(), kit.Rid)
		return nil, ccErr
	}

	if len(instResult.Data.Info) == 0 {
		blog.Errorf("GetSets failed, set not found, filter: %s, instResult: %s, rid: %s", filter, instResult, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommNotFound)
	}

	if instResult.Data.Count != len(setIDs) {
		blog.Errorf("GetSets failed, some setID invalid, input IDs: %+v, valid ,IDs: %+v, rid: %s",
			setIDs, instResult.Data.Info, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_set_ids")
	}

	return instResult.Data.Info, nil
}

func (st *setTemplate) isSyncRequired(kit *rest.Kit, bizID int64, setTemplateID int64, setIDs []int64, isInterrupt bool) (map[int64]bool,
	errors.CCErrorCoder) {

	if len(setIDs) == 0 {
		blog.Errorf("array of set_id is empty, rid: %s", kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, common.BKSetIDField)
	}

	serviceTemplates, err := st.client.CoreService().SetTemplate().ListSetTplRelatedSvcTpl(kit.Ctx, kit.Header, bizID,
		setTemplateID)
	if err != nil {
		blog.Errorf(" list set template and service template related failed, bizID: %d, "+
			"setTemplateID: %d, err: %s, rid: %s", bizID, setTemplateID, err.Error(), kit.Rid)
		return nil, err
	}

	svcTplCnt := int64(len(serviceTemplates))
	svcTplMap := make(map[int64]metadata.ServiceTemplate, svcTplCnt)
	for _, serviceTemplate := range serviceTemplates {
		svcTplMap[serviceTemplate.ID] = serviceTemplate
	}

	moduleFilter := &metadata.QueryCondition{
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
		Fields: []string{
			common.BKSetIDField,
			common.BKModuleIDField,
			common.BKSetTemplateIDField,
			common.BKModuleNameField,
			common.BKServiceTemplateIDField,
		},
		Condition: mapstr.MapStr(map[string]interface{}{
			common.BKSetTemplateIDField: setTemplateID,
			common.BKSetIDField: map[string]interface{}{
				common.BKDBIN: setIDs,
			},
		}),
	}

	modulesInstResult := new(metadata.ResponseModuleInstance)
	if err := st.client.CoreService().Instance().ReadInstanceStruct(kit.Ctx, kit.Header, common.BKInnerObjIDModule,
		moduleFilter, modulesInstResult); err != nil {
		blog.Errorf("list modules failed, bizID: %s, setTemplateID: %s, setIDs: %s, err: %s, rid: %s",
			bizID, setTemplateID, setIDs, err, kit.Rid)
		return nil, err
	}

	if err := modulesInstResult.CCError(); err != nil {
		blog.Errorf("list module http reply failed, bizID: %s, setTemplateID: %s, setIDs: %s, filter: %s, "+
			"reply: %s, rid: %s", bizID, setTemplateID, setIDs, moduleFilter, modulesInstResult, kit.Rid)
		return nil, err
	}

	setModules := make(map[int64][]metadata.ModuleInst, len(modulesInstResult.Data.Info))
	for _, module := range modulesInstResult.Data.Info {
		if _, exist := setModules[module.SetID]; !exist {
			setModules[module.SetID] = make([]metadata.ModuleInst, 0)
		}
		setModules[module.SetID] = append(setModules[module.SetID], module)
	}

	checkResult := make(map[int64]bool, len(setModules))
	for idx, module := range setModules {
		checkResult[idx] = diffModuleServiceTpl(svcTplCnt, svcTplMap, int64(len(module)), module)
		if isInterrupt && checkResult[idx] {
			return checkResult, nil
		}
	}

	return checkResult, nil
}

// diffModuleServiceTpl check different of modules with template in one set
func diffModuleServiceTpl(serviceTplCnt int64, serviceTemplates map[int64]metadata.ServiceTemplate, moduleCnt int64,
	modules []metadata.ModuleInst) bool {
	/*
		depend on logic in func DiffServiceTemplateWithModules
		if the number of the module and the template is not the same, it changed
		if the name of the module and the template is not the same, it changed
		this function only use to check if module and template are the same
	*/

	if serviceTplCnt != moduleCnt {
		return true
	}

	for _, module := range modules {
		template, ok := serviceTemplates[module.ServiceTemplateID]
		if !ok {
			return true
		}
		if template.Name != module.ModuleName {
			return true
		}
	}

	return false
}

func (st *setTemplate) UpdateSetSyncStatus(kit *rest.Kit, setTemplateID int64, setID []int64) ([]metadata.SetTemplateSyncStatus, errors.CCErrorCoder) {
	var setSyncStatus []metadata.SetTemplateSyncStatus

	if setTemplateID == common.SetTemplateIDNotSet {
		blog.V(4).Infof("set not bound with template, setID: %d, rid: %s", setID, kit.Rid)
		return setSyncStatus, nil
	}

	sets, err := st.GetSets(kit, setTemplateID, setID)
	if err != nil {
		blog.Errorf("get sets failed, setID: %d, err: %s, rid: %s", setID, err.Error(), kit.Rid)
		return nil, err
	}

	if len(sets) == 0 {
		blog.Errorf("get sets success but return is empty setID: %d, rid: %s", setID, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsIsInvalid, common.BKSetIDField)
	}

	bizID := sets[0].BizID
	needSyncs, err := st.isSyncRequired(kit, bizID, setTemplateID, setID, false)
	if err != nil {
		blog.Errorf("check sync required failed, templateID: %d, setID: %d, err: %s, rid: %s",
			setTemplateID, setID, err.Error(), kit.Rid)
		return nil, err
	}

	if len(needSyncs) == 0 {
		blog.Errorf("check sync required return empty, templateID: %d, setID: %d, rid: %s",
			setTemplateID, setID, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommInternalServerError)
	}

	taskCond := metadata.ListAPITaskDetail{
		SetID: setID,
		Fields: []string{
			common.CreateTimeField,
			common.LastTimeField,
			common.BKUser,
			common.BKTaskIDField,
			common.BKStatusField,
			common.MetaDataSynchronizeFlagField,
			common.BKInstIDField,
		},
	}
	details, err := st.GetLatestSyncTaskDetail(kit, taskCond)
	if err != nil {
		return setSyncStatus, err
	}

	for _, set := range sets {
		// update sync status
		syncStatus := metadata.SetTemplateSyncStatus{
			SetID:           set.SetID,
			Name:            set.SetName,
			BizID:           set.BizID,
			SetTemplateID:   set.SetTemplateID,
			SupplierAccount: set.SupplierAccount,
			Creator:         kit.User,
			CreateTime:      set.CreateTime,
			LastTime:        set.LastTime,
			TaskID:          "",
			Status:          metadata.SyncStatusFinished,
		}

		if needSyncs[set.SetID] {
			syncStatus.Status = metadata.SyncStatusWaiting
		}

		if _, ok := details[set.SetID]; !ok {
			setSyncStatus = append(setSyncStatus, syncStatus)
			continue
		}

		syncStatus.Creator = details[set.SetID].User
		syncStatus.CreateTime = metadata.Time{Time: details[set.SetID].CreateTime}
		syncStatus.LastTime = metadata.Time{Time: details[set.SetID].LastTime}
		syncStatus.TaskID = details[set.SetID].TaskID

		if !details[set.SetID].Status.IsFinished() {
			syncStatus.Status = metadata.SyncStatusSyncing
		}

		if details[set.SetID].Status.IsFailure() {
			syncStatus.Status = metadata.SyncStatusFailure
		}

		setSyncStatus = append(setSyncStatus, syncStatus)
	}

	err = st.client.CoreService().SetTemplate().UpdateManySetTemplateSyncStatus(kit.Ctx, kit.Header, setSyncStatus)
	if err != nil {
		blog.Errorf("UpdateSetSyncStatus failed, UpdateSetTemplateSyncStatus failed, setID: %d, err: %s, rid: %s", setID, err.Error(), kit.Rid)
		return nil, err
	}

	return setSyncStatus, nil
}

func (st *setTemplate) GetLatestSyncTaskDetail(kit *rest.Kit,
	taskCond metadata.ListAPITaskDetail) (map[int64]*metadata.APITaskDetail, errors.CCErrorCoder) {

	if len(taskCond.SetID) == 0 {
		blog.Errorf("set id is empty, rid: %s", kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrTaskListTaskFail)
	}

	latestTaskResult := make(map[int64]*metadata.APITaskDetail)

	setRelatedTaskFilter := map[string]interface{}{
		"bk_inst_id": map[string]interface{}{common.BKDBIN: taskCond.SetID},
		"flag": common.SyncSetTaskFlag,
	}
	listTaskOption := new(metadata.ListAPITaskLatestRequest)
	listTaskOption.Condition = setRelatedTaskFilter
	listTaskOption.Fields = taskCond.Fields

	listResult, err := st.client.TaskServer().Task().ListLatestTask(kit.Ctx, kit.Header, common.SyncSetTaskName, listTaskOption)
	if err != nil {
		blog.Errorf("list set sync tasks failed, option: %s, err: %v, rid: %s", listTaskOption, err, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrTaskListTaskFail)
	}

	if listResult == nil || len(listResult.Data) == 0 {
		return latestTaskResult, nil
	}

	for _, APITask := range listResult.Data {
		if len(taskCond.Fields) == 0 {
			clearSetSyncTaskDetail(&APITask)
		}

		if APITask.InstID != 0 {
			latestTaskResult[APITask.InstID] = &APITask
		}
	}

	return latestTaskResult, nil
}

func clearSetSyncTaskDetail(detail *metadata.APITaskDetail) {
	detail.Header = util.BuildHeader(common.CCSystemOperatorUserName, common.BKDefaultOwnerID)
	for taskIdx := range detail.Detail {
		subTaskDetail, ok := detail.Detail[taskIdx].Data.(map[string]interface{})
		if !ok {
			blog.Warnf("clearSetSyncTaskDetail expect map[string]interface{}, got unexpected type, data: %+v", detail.Detail[taskIdx].Data)
			detail.Detail[taskIdx].Data = nil
		}
		delete(subTaskDetail, "header")
	}
}

// TriggerCheckSetTemplateSyncingStatus  触发对正在同步中任务的状态改变处理
func (st *setTemplate) TriggerCheckSetTemplateSyncingStatus(kit *rest.Kit, bizID, setTemplateID int64, setID []int64) errors.CCErrorCoder {
	setTempLock := lock.NewLocker(redis.Client())
	key := lock.GetLockKey(lock.CheckSetTemplateSyncFormat, setID)
	locked, err := setTempLock.Lock(key, time.Minute)
	if err != nil {
		blog.Errorf("get sync set template  lock error. set template id: %d, setID: %d, err: %s, rid: %s", setTemplateID, setID, err.Error(), kit.Rid)
		return kit.CCError.CCError(common.CCErrCommRedisOPErr)
	}
	if locked {
		defer setTempLock.Unlock()
		_, err := st.UpdateSetSyncStatus(kit, setTemplateID, setID)
		if err != nil {
			return err
		}

	} else {
		blog.Warnf("skip task, reason not get lock . set template id: %d, setID: %d, rid: %s", setTemplateID, setID, kit.Rid)
	}
	return nil
}

func (st *setTemplate) ListSetTemplateSyncStatus(kit *rest.Kit, bizID int64,
	option metadata.ListSetTemplateSyncStatusOption) (metadata.MultipleSetTemplateSyncStatus, errors.CCErrorCoder) {

	responseInfo := metadata.MultipleSetTemplateSyncStatus{}

	filterTemp := &metadata.QueryCondition{
		Page:      option.Page,
		Condition: mapstr.MapStr{common.BKSetTemplateIDField: option.SetTemplateID, common.BKAppIDField: bizID},
	}
	if len(option.SetIDs) != 0 {
		filterTemp.Condition[common.BKSetIDField] = mapstr.MapStr{common.BKDBIN: option.SetIDs}
	}
	filterTemp.Fields = []string{common.BKSetIDField, common.BKSetNameField, common.BkSupplierAccount}

	var setInfoResp metadata.ResponseSetInstance
	err := st.client.CoreService().Instance().ReadInstanceStruct(kit.Ctx, kit.Header,
		common.BKInnerObjIDSet, filterTemp, &setInfoResp)
	if err != nil {
		blog.ErrorJSON("ListSetTemplateSyncStatus failed, core service find set template failed,"+
			" option: %s, err: %s, rid: %s", filterTemp, err, kit.Rid)
		return metadata.MultipleSetTemplateSyncStatus{}, err
	}
	if err := setInfoResp.CCError(); err != nil {
		blog.ErrorJSON("ListSetTemplateSyncStatus failed, core service find set template http reply failed,"+
			" option: %s, err: %s, rid: %s", filterTemp, err, kit.Rid)
		return metadata.MultipleSetTemplateSyncStatus{}, err
	}

	responseInfo.Count = int64(setInfoResp.Data.Count)
	setIDs := make([]int64, len(setInfoResp.Data.Info))
	responseInfo.Info = make([]metadata.SetTemplateSyncStatus, len(setInfoResp.Data.Info))

	for idx, setInfo := range setInfoResp.Data.Info {
		setIDs[idx] = setInfo.SetID
		// setInfoResp 只返回了部分字段，新加字段注意修改
		responseInfo.Info[idx] = metadata.SetTemplateSyncStatus{
			SetID:           setInfo.SetID,
			Name:            setInfo.SetName,
			BizID:           bizID,
			SetTemplateID:   option.SetTemplateID,
			SupplierAccount: setInfo.SupplierAccount,
		}
	}

	// 使用存在模块
	option.SetIDs = setIDs
	result, err := st.client.CoreService().SetTemplate().ListSetTemplateSyncStatus(kit.Ctx, kit.Header, bizID, option)
	if err != nil {
		blog.ErrorJSON("ListSetTemplateSyncStatus failed, core service search failed, option: %s, err: %s, rid: %s",
			option, err.Error(), kit.Rid)
		return metadata.MultipleSetTemplateSyncStatus{}, err
	}

	setTempSyncMap := make(map[int64]metadata.SetTemplateSyncStatus, len(result.Info))
	// 处理当前需要同步任务的状态
	for _, info := range result.Info {
		setTempSyncMap[info.SetID] = info
		if !info.Status.IsFinished() {
			go func(info metadata.SetTemplateSyncStatus) {
				st.TriggerCheckSetTemplateSyncingStatus(kit.NewKit(), info.BizID, info.SetTemplateID, []int64{info.SetID})
			}(info)
		}

	}
	// 如果在同步表中有数据，使用同步表中的数据
	for idx, row := range responseInfo.Info {
		if newRow, ok := setTempSyncMap[row.SetID]; ok {
			responseInfo.Info[idx] = newRow
		}
	}

	return responseInfo, nil

}

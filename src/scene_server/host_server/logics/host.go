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

package logics

import (
	"strings"

	"configcenter/src/common"
	"configcenter/src/common/auditlog"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/util"
)

func (lgc *Logics) GetHostAttributes(kit *rest.Kit, bizMetaOpt mapstr.MapStr) ([]metadata.Attribute, error) {
	searchOp := mapstr.MapStr{
		common.BKObjIDField: common.BKInnerObjIDHost,
	}
	if bizMetaOpt != nil {
		searchOp.Merge(bizMetaOpt)
	}
	query := &metadata.QueryCondition{
		Condition: searchOp,
	}
	result, err := lgc.CoreAPI.CoreService().Model().ReadModelAttr(kit.Ctx, kit.Header, common.BKInnerObjIDHost, query)
	if err != nil {
		blog.Errorf("GetHostAttributes http do error, err:%s, input:%+v, rid:%s", err.Error(), query, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !result.Result {
		blog.Errorf("GetHostAttributes http response error, err code:%d, err msg:%s, input:%+v, rid:%s", result.Code, result.ErrMsg, query, kit.Rid)
		return nil, kit.CCError.New(result.Code, result.ErrMsg)
	}

	return result.Data.Info, nil
}

func (lgc *Logics) GetHostInstanceDetails(kit *rest.Kit, hostID int64) (map[string]interface{}, string, errors.CCError) {
	// get host details, pre data
	result, err := lgc.CoreAPI.CoreService().Host().GetHostByID(kit.Ctx, kit.Header, hostID)
	if err != nil {
		blog.Errorf("GetHostInstanceDetails http do error, err:%s, input:%+v, rid:%s", err.Error(), hostID, kit.Rid)
		return nil, "", kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !result.Result {
		blog.Errorf("GetHostInstanceDetails http response error, err code:%d, err msg:%s, input:%+v, rid:%s", result.Code, result.ErrMsg, hostID, kit.Rid)
		return nil, "", kit.CCError.New(result.Code, result.ErrMsg)
	}

	hostInfo := result.Data
	if len(hostInfo) == 0 {
		return nil, "", nil
	}
	ip, ok := hostInfo[common.BKHostInnerIPField].(string)
	if !ok {
		blog.Errorf("GetHostInstanceDetails http response format error,convert bk_biz_id to int error, inst:%#v  input:%#v, rid:%s", hostInfo, hostID, kit.Rid)
		return nil, "", kit.CCError.Errorf(common.CCErrCommInstFieldConvertFail, common.BKInnerObjIDHost, common.BKHostInnerIPField, "string", "not string")

	}
	return hostInfo, ip, nil
}

// GetConfigByCond get hosts owned set, module info, where hosts must match condition specify by cond.
func (lgc *Logics) GetConfigByCond(kit *rest.Kit, input metadata.HostModuleRelationRequest) ([]metadata.ModuleHost, errors.CCError) {

	result, err := lgc.CoreAPI.CoreService().Host().GetHostModuleRelation(kit.Ctx, kit.Header, &input)
	if err != nil {
		blog.Errorf("GetConfigByCond http do error, err:%s, input:%+v, rid:%s", err.Error(), input, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !result.Result {
		blog.Errorf("GetConfigByCond http response error, err code:%d, err msg:%s, input:%+v, rid:%s", result.Code, result.ErrMsg, input, kit.Rid)
		return nil, kit.CCError.New(result.Code, result.ErrMsg)
	}

	return result.Data.Info, nil
}

// EnterIP 将机器导入到指定模块或者空闲模块， 已经存在机器，不操作
func (lgc *Logics) EnterIP(kit *rest.Kit, appID, moduleID int64, ip string, cloudID int64, host map[string]interface{}, isIncrement bool) errors.CCError {

	isExist, err := lgc.IsPlatExist(kit, mapstr.MapStr{common.BKCloudIDField: cloudID})
	if nil != err {
		return err
	}
	if !isExist {
		return kit.CCError.Errorf(common.CCErrTopoCloudNotFound)
	}
	ipArr := strings.Split(ip, ",")
	conds := mapstr.MapStr{
		common.BKHostInnerIPField: map[string]interface{}{
			common.BKDBAll:  ipArr,
			common.BKDBSize: len(ipArr),
		},
		common.BKCloudIDField: cloudID,
	}
	hostList, err := lgc.GetHostInfoByConds(kit, conds)
	if nil != err {
		return err
	}

	hostID := int64(0)
	if len(hostList) == 0 {
		//host not exist, add host
		host[common.BKHostInnerIPField] = ip
		host[common.BKCloudIDField] = cloudID
		host["import_from"] = common.HostAddMethodAgent
		defaultFields, hasErr := lgc.getHostFields(kit)
		if nil != hasErr {
			return hasErr
		}
		//补充未填写字段的默认值
		for _, field := range defaultFields {
			_, ok := host[field.PropertyID]
			if !ok {
				if true == util.IsStrProperty(field.PropertyType) {
					host[field.PropertyID] = ""
				} else {
					host[field.PropertyID] = nil
				}
			}
		}

		result, err := lgc.CoreAPI.CoreService().Instance().CreateInstance(kit.Ctx, kit.Header, common.BKInnerObjIDHost, &metadata.CreateModelInstance{Data: host})
		if err != nil {
			blog.Errorf("EnterIP http do error, err:%s, input:%+v, rid:%s", err.Error(), host, kit.Rid)
			return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
		}
		if !result.Result {
			blog.Errorf("EnterIP http response error, err code:%d, err msg:%s, input:%+v, rid:%s", result.Code, result.ErrMsg, host, kit.Rid)
			return kit.CCError.New(result.Code, result.ErrMsg)
		}

		// add audit log for create host.
		audit := auditlog.NewHostAudit(lgc.CoreAPI.CoreService())
		generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditCreate)
		auditLog, err := audit.GenerateAuditLog(generateAuditParameter, hostID, appID, "", nil)
		if err != nil {
			blog.Errorf("generate audit log failed after create host, hostID: %d, appID: %d, err: %v, rid: %s",
				hostID, appID, err, kit.Rid)
			return err
		}

		// save audit log.
		if err := audit.SaveAuditLog(kit, *auditLog); err != nil {
			blog.Errorf("save audit log failed after create host, hostID: %d, appID: %d,err: %v, rid: %s", hostID,
				appID, err, kit.Rid)
			return err
		}

		hostID = int64(result.Data.Created.ID)
	} else if false == isIncrement {
		// Not an additional relationship model
		return nil
	} else {

		hostID, err = util.GetInt64ByInterface(hostList[0][common.BKHostIDField])
		if err != nil {
			blog.Errorf("EnterIP  get hostID error, err:%s,inst:%+v,input:%+v, rid:%s", err.Error(), hostList[0], host, kit.Rid)
			return kit.CCError.Errorf(common.CCErrCommInstFieldConvertFail, common.BKInnerObjIDHost, common.BKHostIDField, "int", err.Error()) // "查询主机信息失败"
		}

		bl, hasErr := lgc.IsHostExistInApp(kit, appID, hostID)
		if nil != hasErr {
			return hasErr

		}
		if false == bl {
			blog.Errorf("Host does not belong to the current application; error, params:{appID:%d, hostID:%d}, rid:%s", appID, hostID, kit.Rid)
			return kit.CCError.Errorf(common.CCErrHostNotINAPPFail, hostID)
		}

	}

	hmAudit := auditlog.NewHostModuleLog(lgc.CoreAPI.CoreService(), kit, []int64{hostID})
	if err := hmAudit.WithPrevious(kit.Ctx); err != nil {
		return err
	}

	params := &metadata.HostsModuleRelation{
		ApplicationID: appID,
		HostID:        []int64{hostID},
		ModuleID:      []int64{moduleID},
		IsIncrement:   isIncrement,
	}
	hmResult, ccErr := lgc.CoreAPI.CoreService().Host().TransferToNormalModule(kit.Ctx, kit.Header, params)
	if ccErr != nil {
		blog.Errorf("Host does not belong to the current application; error, params:{appID:%d, hostID:%d}, err:%s, rid:%s", appID, hostID, err.Error(), kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !hmResult.Result {
		blog.Errorf("transfer host to normal module failed, error params:{appID:%d, hostID:%d}, result:%#v, rid:%s", appID, hostID, hmResult, kit.Rid)
		if len(hmResult.Data) > 0 {
			return kit.CCError.New(int(hmResult.Data[0].Code), hmResult.Data[0].Message)
		}
		return kit.CCError.New(hmResult.Code, hmResult.ErrMsg)
	}

	if err := hmAudit.SaveAudit(kit.Ctx); err != nil {
		return err
	}
	return nil
}

func (lgc *Logics) GetHostInfoByConds(kit *rest.Kit, cond map[string]interface{}) ([]mapstr.MapStr, errors.CCErrorCoder) {
	query := &metadata.QueryInput{
		Condition: cond,
		Start:     0,
		Limit:     common.BKNoLimit,
		Sort:      common.BKHostIDField,
	}

	result, err := lgc.CoreAPI.CoreService().Host().GetHosts(kit.Ctx, kit.Header, query)
	if err != nil {
		blog.Errorf("GetHostInfoByConds GetHosts http do error, err:%s, input:%+v,rid:%s", err.Error(), query, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}
	if err := result.CCError(); err != nil {
		blog.Errorf("GetHostInfoByConds GetHosts http response error, err code:%d, err msg:%s,input:%+v,rid:%s", result.Code, result.ErrMsg, query, kit.Rid)
		return nil, err
	}

	return result.Data.Info, nil
}

// SearchHostInfo search host info by QueryCondition
func (lgc *Logics) SearchHostInfo(kit *rest.Kit, cond metadata.QueryCondition) ([]mapstr.MapStr, errors.CCErrorCoder) {
	query := &metadata.QueryInput{
		Condition: cond.Condition,
		Fields:    strings.Join(cond.Fields, ","),
		Start:     cond.Page.Start,
		Limit:     cond.Page.Limit,
		Sort:      cond.Page.Sort,
	}

	result, err := lgc.CoreAPI.CoreService().Host().GetHosts(kit.Ctx, kit.Header, query)
	if err != nil {
		blog.Errorf("GetHostInfoByConds GetHosts http do error, err:%s, input:%+v,rid:%s", err.Error(), query, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}
	if err := result.CCError(); err != nil {
		blog.Errorf("GetHostInfoByConds GetHosts http response error, err code:%d, err msg:%s,input:%+v,rid:%s", result.Code, result.ErrMsg, query, kit.Rid)
		return nil, err
	}

	return result.Data.Info, nil
}

// HostSearch search host by multiple condition
const (
	SplitFlag      = "##"
	TopoSetName    = "TopSetName"
	TopoModuleName = "TopModuleName"
)

// GetHostIDByCond query hostIDs by condition base on cc_ModuleHostConfig
// available condition fields are bk_supplier_account, bk_biz_id, bk_host_id, bk_module_id, bk_set_id
func (lgc *Logics) GetHostIDByCond(kit *rest.Kit, cond metadata.HostModuleRelationRequest) ([]int64, errors.CCError) {

	cond.Fields = []string{common.BKHostIDField}
	result, err := lgc.CoreAPI.CoreService().Host().GetHostModuleRelation(kit.Ctx, kit.Header, &cond)
	if err != nil {
		blog.Errorf("GetHostIDByCond GetModulesHostConfig http do error, err:%s, input:%+v,rid:%s", err.Error(), cond, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !result.Result {
		blog.Errorf("GetHostIDByCond GetModulesHostConfig http response error, err code:%d, err msg:%s,input:%+v,rid:%s", result.Code, result.ErrMsg, cond, kit.Rid)
		return nil, kit.CCError.New(result.Code, result.ErrMsg)
	}

	hostIDs := make([]int64, 0)
	for _, val := range result.Data.Info {
		hostIDs = append(hostIDs, val.HostID)
	}

	return hostIDs, nil
}

// GetAllHostIDByCond 专用结构， page start 和limit 无效， 获取条件所有满足条件的主机
func (lgc *Logics) GetAllHostIDByCond(kit *rest.Kit, cond metadata.HostModuleRelationRequest) ([]int64, errors.CCError) {
	hostIDs := make([]int64, 0)
	cond.Page.Limit = 2000
	start := 0
	cnt := 0
	cond.Fields = []string{common.BKHostIDField}
	for {
		cond.Page.Start = start
		result, err := lgc.CoreAPI.CoreService().Host().GetHostModuleRelation(kit.Ctx, kit.Header, &cond)
		if err != nil {
			blog.Errorf("GetHostIDByCond GetModulesHostConfig http do error, err:%s, input:%+v,rid:%s", err.Error(), cond, kit.Rid)
			return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
		}
		if !result.Result {
			blog.Errorf("GetHostIDByCond GetModulesHostConfig http response error, err code:%d, err msg:%s,input:%+v,rid:%s", result.Code, result.ErrMsg, cond, kit.Rid)
			return nil, kit.CCError.New(result.Code, result.ErrMsg)
		}

		for _, val := range result.Data.Info {
			hostIDs = append(hostIDs, val.HostID)
		}
		// 当总数大于现在的总数，使用当前返回值的总是为新的总数值
		if cnt < int(result.Data.Count) {
			// 获取条件的数据总数
			cnt = int(result.Data.Count)
		}
		start += cond.Page.Limit
		if start >= cnt {
			break
		}
	}

	return hostIDs, nil
}

// DeleteHostBusinessAttributes delete host business private property
func (lgc *Logics) DeleteHostBusinessAttributes(kit *rest.Kit, hostIDArr []int64, bizID int64) error {

	return nil
}

// GetHostModuleRelation  query host and module relation,
// condition key use appID, moduleID,setID,HostID
func (lgc *Logics) GetHostModuleRelation(kit *rest.Kit, cond metadata.HostModuleRelationRequest) (*metadata.HostConfigData, errors.CCErrorCoder) {

	if cond.Empty() {
		return nil, kit.CCError.CCError(common.CCErrCommHTTPBodyEmpty)
	}

	if cond.Page.IsIllegal() {
		return nil, kit.CCError.CCError(common.CCErrCommPageLimitIsExceeded)
	}

	if len(cond.SetIDArr) > 200 {
		return nil, kit.CCError.CCErrorf(common.CCErrCommXXExceedLimit, "bk_set_ids", 200)
	}

	if len(cond.ModuleIDArr) > 500 {
		return nil, kit.CCError.CCErrorf(common.CCErrCommXXExceedLimit, "bk_module_ids", 500)
	}

	if len(cond.HostIDArr) > 500 {
		return nil, kit.CCError.CCErrorf(common.CCErrCommXXExceedLimit, "bk_host_ids", 500)
	}

	result, err := lgc.CoreAPI.CoreService().Host().GetHostModuleRelation(kit.Ctx, kit.Header, &cond)
	if err != nil {
		blog.Errorf("GetHostModuleRelation http do error, err:%s, input:%+v, rid:%s", err.Error(), cond, kit.Rid)
		return nil, kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}
	if retErr := result.CCError(); retErr != nil {
		blog.Errorf("GetHostModuleRelation http response error, err code:%d, err msg:%s, input:%+v, rid:%s", result.Code, result.ErrMsg, cond, kit.Rid)
		return nil, retErr
	}

	return &result.Data, nil
}

// TransferHostAcrossBusiness  Transfer host across business,
// delete old business  host and module relation
func (lgc *Logics) TransferHostAcrossBusiness(kit *rest.Kit, srcBizID, dstAppID int64, hostID []int64, moduleID []int64) errors.CCError {
	notExistHostIDs, err := lgc.ExistHostIDSInApp(kit, srcBizID, hostID)
	if err != nil {
		blog.Errorf("TransferHostAcrossBusiness IsHostExistInApp err:%s,input:{appID:%d,hostID:%d},rid:%s", err.Error(), srcBizID, hostID, kit.Rid)
		return err
	}
	if len(notExistHostIDs) > 0 {
		blog.Errorf("TransferHostAcrossBusiness Host does not belong to the current application; error, params:{appID:%d, hostID:%+v}, rid:%s", srcBizID, notExistHostIDs, kit.Rid)
		return kit.CCError.Errorf(common.CCErrHostNotINAPP, notExistHostIDs)
	}

	audit := auditlog.NewHostModuleLog(lgc.CoreAPI.CoreService(), kit, hostID)
	if err := audit.WithPrevious(kit.Ctx); err != nil {
		blog.Errorf("TransferHostAcrossBusiness, get prev module host config failed, err: %v,hostID:%d,oldbizID:%d,appID:%d, moduleID:%#v,rid:%s", err, hostID, srcBizID, dstAppID, moduleID, kit.Rid)
		return kit.CCError.Errorf(common.CCErrCommResourceInitFailed, "audit server")
	}
	conf := &metadata.TransferHostsCrossBusinessRequest{SrcApplicationID: srcBizID, HostIDArr: hostID, DstApplicationID: dstAppID, DstModuleIDArr: moduleID}
	delRet, doErr := lgc.CoreAPI.CoreService().Host().TransferToAnotherBusiness(kit.Ctx, kit.Header, conf)
	if doErr != nil {
		blog.Errorf("TransferHostAcrossBusiness http do error, err:%s, input:%+v, rid:%s", doErr.Error(), conf, kit.Rid)
		return kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !delRet.Result {
		blog.Errorf("TransferHostAcrossBusiness http response error, err code:%d, err msg:%s, input:%#v, rid:%s", delRet.Code, delRet.ErrMsg, conf, kit.Rid)
		return kit.CCError.New(delRet.Code, delRet.ErrMsg)
	}

	if err := audit.SaveAudit(kit.Ctx); err != nil {
		blog.Errorf("TransferHostAcrossBusiness, get prev module host config failed, err: %v,hostID:%d,oldbizID:%d,appID:%d, moduleID:%#v,rid:%s", err, hostID, srcBizID, dstAppID, moduleID, kit.Rid)
		return kit.CCError.Errorf(common.CCErrCommResourceInitFailed, "audit server")

	}

	return nil
}

// TransferHostAcrossBusinessPreview transfer host across business, service instance, host apply preview
func (lgc *Logics) TransferHostAcrossBusinessPreview(kit *rest.Kit, srcBizID, dstBizID int64, hostIDs []int64,
	moduleIDs []int64) ([]metadata.HostTransferPreview, errors.CCError) {

	// get src biz idle module
	srcModuleCond := map[string]interface{}{
		common.BKAppIDField:   srcBizID,
		common.BKDefaultField: common.DefaultResModuleFlag,
	}

	srcModuleID, err := lgc.GetResourcePoolModuleID(kit, srcModuleCond)
	if err != nil {
		blog.Errorf("transfer host across biz preview, get src biz(%d) idle module id failed, err: %v, rid: %s", srcBizID, err, kit.Rid)
		return nil, err
	}

	// check if hosts are in the src biz idle module
	errHostID, err := lgc.notExistAppModuleHost(kit, srcBizID, []int64{srcModuleID}, hostIDs)
	if err != nil {
		blog.Errorf("transfer host across biz preview, check hosts in src biz idle module failed, err: %v, bizID: %d, hostIDs: %+v, rid: %s", err, srcBizID, hostIDs, kit.Rid)
		return nil, err
	}

	if len(errHostID) > 0 {
		errHostIP := lgc.convertHostIDToHostIP(kit, errHostID)
		blog.Errorf("transfer host across biz preview, has host not belong to idle module, bizID: %d, err host inner ip:%#v, rid:%s", srcBizID, errHostIP, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrNotBelongToIdleModule, util.PrettyIPStr(errHostIP))
	}

	// check if dest modules are in dest biz and do not contain both inner and outer modules
	moduleIDs = util.IntArrayUnique(moduleIDs)
	query := &metadata.QueryCondition{
		Page:   metadata.BasePage{Limit: common.BKNoLimit},
		Fields: []string{common.BKModuleIDField, common.BKDefaultField, common.BKServiceTemplateIDField, common.HostApplyEnabledField},
		Condition: map[string]interface{}{
			common.BKAppIDField: dstBizID,
			common.BKModuleIDField: map[string]interface{}{
				common.BKDBIN: moduleIDs,
			},
		},
	}

	destModuleRes, err := lgc.CoreAPI.CoreService().Instance().ReadInstance(kit.Ctx, kit.Header, common.BKInnerObjIDModule, query)
	if err != nil {
		blog.Errorf("transfer host across biz preview, valid dest modules failed, err: %v, bizID: %d, moduleIDs: %+v, rid: %s", err, dstBizID, moduleIDs, kit.Rid)
		return nil, err
	}
	if !destModuleRes.Result {
		blog.Errorf("transfer host across biz preview, valid dest modules failed, err: %s, bizID: %d, moduleIDs: %+v, rid: %s", destModuleRes.ErrMsg, dstBizID, moduleIDs, kit.Rid)
		return nil, destModuleRes.CCError()
	}

	if len(destModuleRes.Data.Info) != len(moduleIDs) {
		blog.Errorf("transfer host across biz preview, not all modules are in the dest biz, moduleIDs: %+v, rid: %s", moduleIDs, kit.Rid)
		return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_module_ids")
	}

	hasInnerModule := false
	serviceTemplateIDs := make([]int64, 0)
	hostApplyModuleIDs := make([]int64, 0)
	moduleServiceTemplateMap := make(map[int64]int64)
	for _, module := range destModuleRes.Data.Info {
		def, err := util.GetIntByInterface(module[common.BKDefaultField])
		if err != nil {
			blog.ErrorJSON("transfer host across biz preview, module(%s) has invalid default field, rid: %s", module, kit.Rid)
			return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_module_ids")
		}

		if def == common.DefaultResModuleFlag || def == common.DefaultFaultModuleFlag || def == common.DefaultRecycleModuleFlag {
			hasInnerModule = true
		} else if hasInnerModule {
			blog.ErrorJSON("transfer host across biz preview, transfer to both inner and outer modules, rid: %s", kit.Rid)
			return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_module_ids")
		}

		moduleID, err := util.GetInt64ByInterface(module[common.BKModuleIDField])
		if err != nil {
			blog.ErrorJSON("transfer host across biz preview, module(%s) has invalid module id field, rid: %s", module, kit.Rid)
			return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_module_ids")
		}

		serviceTemplateID, err := util.GetInt64ByInterface(module[common.BKServiceTemplateIDField])
		if err != nil {
			blog.ErrorJSON("transfer host across biz preview, module(%s) has invalid service template id field, rid: %s", module, kit.Rid)
			return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_module_ids")
		}

		if serviceTemplateID != common.ServiceTemplateIDNotSet {
			serviceTemplateIDs = append(serviceTemplateIDs, serviceTemplateID)
			moduleServiceTemplateMap[moduleID] = serviceTemplateID
		}

		enabled, ok := module[common.HostApplyEnabledField].(bool)
		if !ok {
			blog.ErrorJSON("transfer host across biz preview, module(%s) has invalid host_apply_enabled field, rid: %s", module, kit.Rid)
			return nil, kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, "bk_module_ids")
		}

		if enabled {
			hostApplyModuleIDs = append(hostApplyModuleIDs, moduleID)
		}
	}

	// get service template related to modules
	serviceTemplateDetails, err := lgc.CoreAPI.CoreService().Process().ListServiceTemplateDetail(kit.Ctx, kit.Header, dstBizID, serviceTemplateIDs...)
	if err != nil {
		blog.Errorf("transfer host across biz preview, get service template failed, err: %s, ids: %+v, rid: %s", dstBizID, err.Error(), serviceTemplateIDs, kit.Rid)
		return nil, err
	}

	serviceTemplateMap := make(map[int64]*metadata.ServiceTemplateDetail)
	for _, templateDetail := range serviceTemplateDetails.Info {
		serviceTemplateMap[templateDetail.ServiceTemplate.ID] = &templateDetail
	}

	toAddToModules := make([]metadata.AddToModuleInfo, len(moduleIDs))
	for index, moduleID := range moduleIDs {
		toAddToModules[index] = metadata.AddToModuleInfo{
			ModuleID:        moduleID,
			ServiceTemplate: serviceTemplateMap[moduleServiceTemplateMap[moduleID]],
		}
	}

	// generate host apply plans
	ruleOption := metadata.ListHostApplyRuleOption{
		ModuleIDs: hostApplyModuleIDs,
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
		},
	}
	rules, err := lgc.CoreAPI.CoreService().HostApplyRule().ListHostApplyRule(kit.Ctx, kit.Header, dstBizID, ruleOption)
	if err != nil {
		blog.Errorf("transfer host across biz preview, get host apply rules failed, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, err
	}

	hostModules := make([]metadata.Host2Modules, len(hostIDs))
	for index, hostID := range hostIDs {
		hostModules[index] = metadata.Host2Modules{
			HostID:    hostID,
			ModuleIDs: hostApplyModuleIDs,
		}
	}

	planOption := metadata.HostApplyPlanOption{
		Rules:       rules.Info,
		HostModules: hostModules,
	}

	hostApplyPlanResult, err := lgc.CoreAPI.CoreService().HostApplyRule().GenerateApplyPlan(kit.Ctx, kit.Header, dstBizID, planOption)
	if err != nil {
		blog.ErrorJSON("transfer host across biz preview, generate host apply rules failed, err: %s, rid: %s", err.Error(), kit.Rid)
		return nil, err
	}
	hostApplyPlanMap := make(map[int64]metadata.OneHostApplyPlan)
	for _, item := range hostApplyPlanResult.Plans {
		hostApplyPlanMap[item.HostID] = item
	}

	previews := make([]metadata.HostTransferPreview, 0)
	for _, hostID := range hostIDs {
		preview := metadata.HostTransferPreview{
			HostID:       hostID,
			FinalModules: moduleIDs,
			ToRemoveFromModules: []metadata.RemoveFromModuleInfo{{
				ModuleID:         srcModuleID,
				ServiceInstances: make([]metadata.ServiceInstance, 0),
			}},
			ToAddToModules: toAddToModules,
			HostApplyPlan:  hostApplyPlanMap[hostID],
		}
		previews = append(previews, preview)
	}
	return previews, nil
}

// DeleteHostFromBusiness  delete host from business,
func (lgc *Logics) DeleteHostFromBusiness(kit *rest.Kit, bizID int64, hostIDArr []int64) ([]metadata.ExceptionResult, errors.CCError) {
	// ready audit log of delete host.
	audit := auditlog.NewHostAudit(lgc.CoreAPI.CoreService())
	logContentMap := make(map[int64]*metadata.AuditLog, 0)
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(kit, metadata.AuditDelete)
	for _, hostID := range hostIDArr {
		var err error
		logContentMap[hostID], err = audit.GenerateAuditLog(generateAuditParameter, hostID, bizID, "", nil)
		if err != nil {
			blog.Errorf("generate host audit log failed before delete host, hostID: %d, bizID: %d, err: %v, rid: %s", hostID, bizID, err, kit.Rid)
			return nil, err
		}
	}

	// to delete host.
	input := &metadata.DeleteHostRequest{
		ApplicationID: bizID,
		HostIDArr:     hostIDArr,
	}
	result, err := lgc.CoreAPI.CoreService().Host().DeleteHostFromSystem(kit.Ctx, kit.Header, input)
	if err != nil {
		blog.Errorf("TransferHostAcrossBusiness DeleteHost error, err: %v,hostID:%#v,appID:%d,rid:%s", err, hostIDArr, bizID, kit.Rid)
		return nil, kit.CCError.Error(common.CCErrCommHTTPDoRequestFailed)
	}
	if !result.Result {
		blog.Errorf("TransferHostAcrossBusiness DeleteHost failed, err: %v,hostID:%#v,appID:%d,rid:%s", err, hostIDArr, bizID, kit.Rid)
		return nil, kit.CCError.New(result.Code, result.ErrMsg)
	}

	// to save audit log.
	logContents := make([]metadata.AuditLog, len(logContentMap))
	index := 0
	for _, item := range logContentMap {
		logContents[index] = *item
		index++
	}

	if len(logContents) > 0 {
		if err := audit.SaveAuditLog(kit, logContents...); err != nil {
			blog.ErrorJSON("delete host in batch, but add host audit log failed, err: %s, rid: %s",
				err, kit.Rid)
			return nil, kit.CCError.Error(common.CCErrAuditSaveLogFailed)
		}

	}
	return nil, nil
}

// CloneHostProperty clone host info and host and module relation in same application
func (lgc *Logics) CloneHostProperty(kit *rest.Kit, appID int64, srcHostID int64, dstHostID int64) errors.CCErrorCoder {

	// source host belong app
	ok, err := lgc.IsHostExistInApp(kit, appID, srcHostID)
	if err != nil {
		blog.Errorf("IsHostExistInApp error. err:%s, params:{appID:%d, hostID:%d}, rid:%s", err.Error(), srcHostID, kit.Rid)
		return err
	}
	if !ok {
		blog.Errorf("Host does not belong to the current application; error, params:{appID:%d, hostID:%d}, rid:%s", appID, srcHostID, kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrHostNotINAPPFail, srcHostID)
	}

	// destination host belong app
	ok, err = lgc.IsHostExistInApp(kit, appID, dstHostID)
	if err != nil {
		blog.Errorf("IsHostExistInApp error. err:%s, params:{appID:%d, hostID:%d}, rid:%s", err.Error(), dstHostID, kit.Rid)
		return err
	}
	if !ok {
		blog.Errorf("Host does not belong to the current application; error, params:{appID:%d, hostID:%d}, rid:%s", appID, dstHostID, kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrHostNotINAPPFail, dstHostID)
	}

	hostInfoArr, err := lgc.GetHostInfoByConds(kit, map[string]interface{}{common.BKHostIDField: srcHostID})
	if err != nil {
		return err
	}
	if len(hostInfoArr) == 0 {
		blog.Errorf("host not found. hostID:%s, rid:%s", srcHostID, kit.Rid)
		return kit.CCError.CCErrorf(common.CCErrHostNotFound)
	}
	srcHostInfo := hostInfoArr[0]

	delete(srcHostInfo, common.BKHostIDField)
	delete(srcHostInfo, common.CreateTimeField)
	delete(srcHostInfo, common.BKHostInnerIPField)
	delete(srcHostInfo, common.BKHostOuterIPField)
	delete(srcHostInfo, common.BKAssetIDField)
	delete(srcHostInfo, common.BKSNField)
	delete(srcHostInfo, common.BKImportFrom)

	// get source host and module relation
	hostModuleRelationCond := metadata.HostModuleRelationRequest{
		ApplicationID: appID,
		HostIDArr:     []int64{srcHostID},
		Page: metadata.BasePage{
			Limit: common.BKNoLimit,
			Start: 0,
		},
	}
	relationArr, err := lgc.GetHostModuleRelation(kit, hostModuleRelationCond)
	if err != nil {
		return err
	}
	var moduleIDArr []int64
	for _, relation := range relationArr.Info {
		moduleIDArr = append(moduleIDArr, relation.ModuleID)
	}

	exist, err := lgc.ExistInnerModule(kit, moduleIDArr)
	if err != nil {
		return err
	}
	if exist {
		if len(moduleIDArr) != 1 {
			return kit.CCError.CCErrorf(common.CCErrHostModuleIDNotFoundORHasMultipleInnerModuleIDFailed)
		}
		dstModuleHostRelation := &metadata.TransferHostToInnerModule{
			ApplicationID: appID,
			HostID:        []int64{dstHostID},
			ModuleID:      moduleIDArr[0],
		}
		relationRet, doErr := lgc.CoreAPI.CoreService().Host().TransferToInnerModule(kit.Ctx, kit.Header, dstModuleHostRelation)
		if doErr != nil {
			blog.ErrorJSON("CloneHostProperty UpdateInstance error. err: %s,condition:%s,rid:%s", doErr, relationRet, kit.Rid)
			return kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
		}
		if err := relationRet.CCError(); err != nil {
			return err
		}
	} else {
		// destination host new module relation
		dstModuleHostRelation := &metadata.HostsModuleRelation{
			ApplicationID: appID,
			HostID:        []int64{dstHostID},
			ModuleID:      moduleIDArr,
			IsIncrement:   false,
		}
		relationRet, doErr := lgc.CoreAPI.CoreService().Host().TransferToNormalModule(kit.Ctx, kit.Header, dstModuleHostRelation)
		if doErr != nil {
			blog.ErrorJSON("CloneHostProperty UpdateInstance error. err: %s,condition:%s,rid:%s", doErr, relationRet, kit.Rid)
			return kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
		}
		if err := relationRet.CCError(); err != nil {
			return err
		}
	}

	input := &metadata.UpdateOption{
		Data: srcHostInfo,
		Condition: mapstr.MapStr{
			common.BKHostIDField: dstHostID,
		},
	}
	result, doErr := lgc.CoreAPI.CoreService().Instance().UpdateInstance(kit.Ctx, kit.Header, common.BKInnerObjIDHost, input)
	if doErr != nil {
		blog.ErrorJSON("CloneHostProperty UpdateInstance error. err: %s,condition:%s,rid:%s", doErr, input, kit.Rid)
		return kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed)
	}
	if err := result.CCError(); err != nil {
		blog.ErrorJSON("CloneHostProperty UpdateInstance  replay error. err: %s,condition:%s,rid:%s", err, input, kit.Rid)
		return err
	}

	return nil
}

// IPCloudToHost get host id by ip and cloud
func (lgc *Logics) IPCloudToHost(kit *rest.Kit, ip string, cloudID int64) (HostMap mapstr.MapStr, hostID int64, err errors.CCErrorCoder) {
	// FIXME there must be a better ip to hostID solution
	ipArr := strings.Split(ip, ",")
	condition := mapstr.MapStr{
		common.BKHostInnerIPField: map[string]interface{}{
			common.BKDBAll:  ipArr,
			common.BKDBSize: len(ipArr),
		},
		common.BKCloudIDField: cloudID,
	}

	hostInfoArr, err := lgc.GetHostInfoByConds(kit, condition)
	if err != nil {
		blog.ErrorJSON("IPCloudToHost GetHostInfoByConds error. err:%s, conditon:%s, rid:%s", err.Error(), condition, kit.Rid)
		return nil, 0, err
	}
	if len(hostInfoArr) == 0 {
		return nil, 0, nil
	}

	hostID, convErr := hostInfoArr[0].Int64(common.BKHostIDField)
	if nil != convErr {
		blog.ErrorJSON("IPCloudToHost bk_host_id field not found hostMap:%s ip:%s, cloudID:%s,rid:%s", hostInfoArr, ip, cloudID, kit.Rid)
		return nil, 0, kit.CCError.CCErrorf(common.CCErrCommInstFieldConvertFail, common.BKInnerObjIDHost, common.BKHostIDField, "int", convErr.Error())
	}

	return hostInfoArr[0], hostID, nil
}

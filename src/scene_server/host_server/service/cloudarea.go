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
	authmeta "configcenter/src/ac/meta"
	"configcenter/src/common"
	"configcenter/src/common/auditlog"
	"configcenter/src/common/auth"
	"configcenter/src/common/blog"
	"configcenter/src/common/errors"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	meta "configcenter/src/common/metadata"
	"configcenter/src/common/paraparse"
	"configcenter/src/common/util"
)

// FindManyCloudArea  find cloud area list
func (s *Service) FindManyCloudArea(ctx *rest.Contexts) {
	rid := ctx.Kit.Rid
	input := new(metadata.CloudAreaSearchParam)
	if err := ctx.DecodeInto(&input); nil != err {
		ctx.RespAutoError(err)
		return
	}

	// set default limit
	if input.Page.Limit == 0 {
		input.Page.Limit = common.BKMaxPageSize
	}
	if input.Page.IsIllegal() {
		blog.Errorf("FindManyCloudArea failed, parse plat page illegal, input:%#v,rid:%s", input, rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommParamsInvalid))
		return
	}

	// set default sort
	if input.Page.Sort == "" {
		input.Page.Sort = "-" + common.CreateTimeField
	}

	// if fuzzy search, change the string query to regexp
	if input.IsFuzzy == true {
		for k, v := range input.Condition {
			field, ok := v.(string)
			if ok {
				input.Condition[k] = mapstr.MapStr{
					common.BKDBLIKE: params.SpecialCharChange(field),
					"$options":      "i",
				}
			}
		}
	}

	query := &metadata.QueryCondition{
		Fields:    input.Fields,
		Condition: input.Condition,
		Page:      input.Page,
	}

	res, err := s.CoreAPI.CoreService().Instance().ReadInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDPlat, query)
	if nil != err {
		blog.Errorf("FindManyCloudArea http do error: %v query:%#v,rid:%s", err, query, rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if false == res.Result {
		blog.Errorf("FindManyCloudArea http reply error.  query:%#v, err code:%d, err msg:%s, rid:%s", query, res.Code, res.ErrMsg, rid)
		ctx.RespAutoError(res.CCError())
		return
	}

	// 查询云区域时附带云同步任务ID信息
	if input.SyncTaskIDs {
		err = s.addPlatSyncTaskIDs(ctx, &res.Data.Info)
		if err != nil {
			blog.ErrorJSON("FindManyCloudArea failed, addPlatSyncTaskIDs err: %v, rid: %s", err, ctx.Kit.Rid)
			ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrHostFindManyCloudAreaAddSyncTaskIDsFieldFail))
			return
		}
	}

	ctx.RespEntity(map[string]interface{}{
		"info":  res.Data.Info,
		"count": res.Data.Count,
	})
}

// CreatePlatBatch create plat instance in batch
func (s *Service) CreatePlatBatch(ctx *rest.Contexts) {

	input := struct {
		Data []mapstr.MapStr `json:"data"`
	}{}

	if err := ctx.DecodeInto(&input); nil != err {
		ctx.RespAutoError(err)
		return
	}

	if len(input.Data) == 0 {
		blog.Errorf("CreatePlat , input is empty, rid:%s", ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPBodyEmpty))
		return
	}

	user := util.GetUser(ctx.Request.Request.Header)
	for i := range input.Data {
		input.Data[i][common.BKCreator] = user
		input.Data[i][common.BKLastEditor] = user
	}

	instInfo := &meta.CreateManyModelInstance{
		Datas: input.Data,
	}

	result := make([]metadata.CreateManyCloudAreaElem, len(input.Data))
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		res, err := s.CoreAPI.CoreService().Instance().CreateManyInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDPlat, instInfo)
		if nil != err {
			blog.Errorf("CreatePlatBatch failed, CreateManyInstance error: %s, input:%+v,rid:%s", err.Error(), input, ctx.Kit.Rid)
			return ctx.Kit.CCError.CCError(common.CCErrTopoInstCreateFailed)
		}

		if false == res.Result {
			blog.Errorf("CreatePlatBatch failed, CreateManyInstance error.err code:%d,err msg:%s,input:%+v,rid:%s", res.Code, res.ErrMsg, input, ctx.Kit.Rid)
			return errors.New(res.Code, res.ErrMsg)
		}

		if len(res.Data.Repeated) > 0 {
			blog.Errorf("CreatePlatBatch failed, err:#v,input:%+v,rid:%s", res.Data.Repeated, input, ctx.Kit.Rid)
			return ctx.Kit.CCError.CCError(common.CCErrCommDuplicateItem)
		}

		if len(res.Data.Exceptions) > 0 {
			blog.Errorf("CreatePlatBatch failed, err:#v,input:%+v,rid:%s", res.Data.Exceptions, input, ctx.Kit.Rid)
			return ctx.Kit.CCError.New(int(res.Data.Exceptions[0].Code), res.Data.Exceptions[0].Message)
		}

		if len(res.Data.Created) == 0 {
			blog.Errorf("CreatePlatBatch failed, no plat was found,input:%+v,rid:%s", input, ctx.Kit.Rid)
			return ctx.Kit.CCError.CCError(common.CCErrTopoCloudNotFound)
		}

		platIDs := make([]int64, len(res.Data.Created))
		for i, created := range res.Data.Created {
			platIDs[i] = int64(created.ID)
			result[i] = metadata.CreateManyCloudAreaElem{
				CloudID: int64(created.ID),
			}
		}

		// generate audit log.
		audit := auditlog.NewCloudAreaAuditLog(s.CoreAPI.CoreService())
		generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(ctx.Kit, metadata.AuditCreate)
		logs, err := audit.GenerateAuditLog(generateAuditParameter, platIDs)
		if err != nil {
			blog.Errorf("generate audit log failed after create cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
			return err
		}

		// save audit log.
		if err := audit.SaveAuditLog(ctx.Kit, logs...); err != nil {
			blog.Errorf("save audit log failed after create cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
			return err
		}

		// register cloud area resource creator action to iam
		if auth.EnableAuthorize() {
			iamInstances := make([]metadata.IamInstance, len(res.Data.Created))
			for index, created := range res.Data.Created {
				iamInstances[index] = metadata.IamInstance{
					ID:   strconv.FormatUint(created.ID, 10),
					Name: util.GetStrByInterface(input.Data[created.OriginIndex][common.BKCloudNameField]),
				}
			}
			iamInstancesWithCreator := metadata.IamInstancesWithCreator{
				IamInstances: metadata.IamInstances{
					Type:      string(iam.SysCloudArea),
					Instances: iamInstances,
				},
				Creator: user,
			}
			_, err = s.AuthManager.Authorizer.BatchRegisterResourceCreatorAction(ctx.Kit.Ctx, ctx.Kit.Header, iamInstancesWithCreator)
			if err != nil {
				blog.Errorf("register created cloud area to iam failed, err: %s, rid: %s", err, ctx.Kit.Rid)
				return err
			}
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}

	ctx.RespEntity(result)

}

// CreatePlat create a plat instance
// available fields for body are last_time, bk_cloud_name, bk_supplier_account, bk_cloud_id, create_time
// {"bk_cloud_name": "云区域", "bk_supplier_account": 0}
func (s *Service) CreatePlat(ctx *rest.Contexts) {
	input := make(map[string]interface{})
	if err := ctx.DecodeInto(&input); nil != err {
		ctx.RespAutoError(err)
		return
	}

	user := util.GetUser(ctx.Request.Request.Header)
	input[common.BKCreator] = user
	input[common.BKLastEditor] = user

	instInfo := &meta.CreateModelInstance{
		Data: mapstr.NewFromMap(input),
	}

	var res *metadata.CreatedOneOptionResult
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		var err error
		res, err = s.CoreAPI.CoreService().Instance().CreateInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDPlat, instInfo)
		if nil != err {
			blog.Errorf("CreatePlat error: %s, input:%+v,rid:%s", err.Error(), input, ctx.Kit.Rid)
			return ctx.Kit.CCError.CCError(common.CCErrTopoInstCreateFailed)
		}

		if false == res.Result {
			blog.Errorf("GetPlat error.err code:%d,err msg:%s,input:%+v,rid:%s", res.Code, res.ErrMsg, input, ctx.Kit.Rid)
			return errors.New(res.Code, res.ErrMsg)
		}

		platID := int64(res.Data.Created.ID)

		// generate audit log.
		audit := auditlog.NewCloudAreaAuditLog(s.CoreAPI.CoreService())
		generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(ctx.Kit, metadata.AuditCreate)
		logs, err := audit.GenerateAuditLog(generateAuditParameter, []int64{platID})
		if err != nil {
			blog.Errorf("generate audit log failed after create cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
			return err
		}

		// save audit log.
		if err := audit.SaveAuditLog(ctx.Kit, logs...); err != nil {
			blog.Errorf("save audit log failed after create cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
			return err
		}

		// register cloud area resource creator action to iam
		if auth.EnableAuthorize() {
			iamInstance := metadata.IamInstancesWithCreator{
				IamInstances: metadata.IamInstances{
					Type: string(iam.SysCloudArea),
					Instances: []metadata.IamInstance{{
						ID:   strconv.FormatInt(platID, 10),
						Name: util.GetStrByInterface(input[common.BKCloudNameField]),
					}},
				},
				Creator: user,
			}
			_, err = s.AuthManager.Authorizer.BatchRegisterResourceCreatorAction(ctx.Kit.Ctx, ctx.Kit.Header, iamInstance)
			if err != nil {
				blog.Errorf("register created cloud area to iam failed, err: %s, rid: %s", err, ctx.Kit.Rid)
				return err
			}
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}

	ctx.RespEntity(res.Data)

}

func (s *Service) DeletePlat(ctx *rest.Contexts) {

	platID, convErr := util.GetInt64ByInterface(ctx.Request.PathParameter(common.BKCloudIDField))
	if nil != convErr {
		blog.Errorf("the platID is invalid, error info is %s, input:%s.rid:%s", convErr.Error(), platID, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, convErr.Error()))
		return
	}
	if 0 == platID {
		blog.Errorf("DelPlat failed, can't delete default cloud area, input:%+v,rid:%s", platID, ctx.Kit.Rid)
		// can't delete default cloud area
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrDeleteDefaultCloudAreaFail))
		return
	}

	params := new(meta.QueryInput)
	params.Fields = common.BKHostIDField
	params.Condition = map[string]interface{}{
		common.BKCloudIDField: platID,
	}

	hostRes, err := s.CoreAPI.CoreService().Host().GetHosts(ctx.Kit.Ctx, ctx.Kit.Header, params)
	if nil != err {
		blog.Errorf("DelPlat search host error: %s, input:%+v,rid:%s", err.Error(), platID, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrHostGetFail))
		return
	}
	if !hostRes.Result {
		blog.Errorf("DelPlat search host http response error.err code:%d,err msg:%s, input:%+v,rid:%s", hostRes.Code, hostRes.ErrMsg, platID, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrHostGetFail))
		return
	}

	// only empty plat could be delete
	if 0 < hostRes.Data.Count {
		blog.Errorf("DelPlat plat [%d] has host data, can not delete,rid:%s", platID, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrTopoHasHostCheckFailed))
		return
	}

	// auth: check authorization
	if err := s.AuthManager.AuthorizeByPlatIDs(ctx.Kit.Ctx, ctx.Kit.Header, authmeta.Delete, platID); err != nil {
		blog.Errorf("check delete plat authorization failed, plat: %d, err: %v, rid: %s", platID, err, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommAuthorizeFailed))
		return
	}

	// generate audit log.
	audit := auditlog.NewCloudAreaAuditLog(s.CoreAPI.CoreService())
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(ctx.Kit, metadata.AuditDelete)
	logs, err := audit.GenerateAuditLog(generateAuditParameter, []int64{platID})
	if err != nil {
		blog.Errorf("generate audit log failed before delete cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	delCond := &meta.DeleteOption{
		Condition: mapstr.MapStr{common.BKCloudIDField: platID},
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		res, err := s.CoreAPI.CoreService().Instance().DeleteInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDPlat, delCond)
		if nil != err {
			blog.Errorf("DelPlat do error: %v, input:%d,rid:%s", err, platID, ctx.Kit.Rid)
			return ctx.Kit.CCError.Errorf(common.CCErrTopoInstDeleteFailed)
		}

		if false == res.Result {
			blog.Errorf("DelPlat http response error. err code:%d,err msg:%s,input:%s,rid:%s", res.Code, res.ErrMsg, platID, ctx.Kit.Rid)
			return res.CCError()
		}

		// save audit log.
		if err := audit.SaveAuditLog(ctx.Kit, logs...); err != nil {
			blog.Errorf("save audit log failed after delete cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
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

func (s *Service) UpdatePlat(ctx *rest.Contexts) {

	// parse platID from url
	platIDStr := ctx.Request.PathParameter(common.BKCloudIDField)
	platID, err := util.GetInt64ByInterface(platIDStr)
	if nil != err {
		blog.Infof("UpdatePlat failed, parse platID failed, platID: %s, err: %s, rid:%s", platIDStr, err.Error(), ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKCloudIDField))
		return
	}
	if 0 == platID {
		blog.Infof("UpdatePlat failed, update built in cloud area forbidden, platID:%+v, rid:%s", platID, ctx.Kit.Rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrTopoUpdateBuiltInCloudForbidden))
		return
	}

	// decode request body
	input := struct {
		CloudName   string `json:"bk_cloud_name"`
		CloudVendor string `json:"bk_cloud_vendor"`
		Region      string `json:"bk_region"`
	}{}

	if err := ctx.DecodeInto(&input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	// update plat
	user := ctx.Kit.User

	toUpdate := mapstr.MapStr{
		common.BKLastEditor: user,
	}

	if len(input.CloudVendor) != 0 {
		toUpdate[common.BKCloudVendor] = input.CloudVendor
	}

	if len(input.Region) != 0 {
		toUpdate[common.BKRegion] = input.Region
	}

	if len(input.CloudName) != 0 {
		toUpdate[common.BKCloudNameField] = input.CloudName
	}

	updateOption := &meta.UpdateOption{
		Data: toUpdate,
		Condition: map[string]interface{}{
			common.BKCloudIDField: platID,
		},
	}

	// generate audit log.
	audit := auditlog.NewCloudAreaAuditLog(s.CoreAPI.CoreService())
	generateAuditParameter := auditlog.NewGenerateAuditCommonParameter(ctx.Kit, metadata.AuditUpdate).WithUpdateFields(toUpdate)
	logs, err := audit.GenerateAuditLog(generateAuditParameter, []int64{platID})
	if err != nil {
		blog.Errorf("generate audit log failed before update cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
		ctx.RespAutoError(err)
		return
	}

	// to update.
	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		res, err := s.CoreAPI.CoreService().Instance().UpdateInstance(ctx.Kit.Ctx, ctx.Kit.Header, common.BKInnerObjIDPlat, updateOption)
		if nil != err {
			blog.ErrorJSON("UpdatePlat failed, UpdateInstance failed, input:%s, err:%s, rid:%s", updateOption, err.Error(), ctx.Kit.Rid)
			return ctx.Kit.CCError.Errorf(common.CCErrTopoInstDeleteFailed)
		}
		if res.Result == false || res.Code != 0 {
			blog.ErrorJSON("UpdatePlat failed, UpdateInstance failed, input:%s, response:%s, rid:%s", updateOption, res, ctx.Kit.Rid)
			return errors.New(res.Code, res.ErrMsg)
		}

		// save audit log.
		if err := audit.SaveAuditLog(ctx.Kit, logs...); err != nil {
			blog.Errorf("save audit log failed after update cloud area, err: %v, rid: %s", err, ctx.Kit.Rid)
			return err
		}

		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}

	// response success
	ctx.RespEntity(nil)
}

func (s *Service) UpdateHostCloudAreaField(ctx *rest.Contexts) {
	rid := ctx.Kit.Rid
	// decode request body
	input := metadata.UpdateHostCloudAreaFieldOption{}
	if err := ctx.DecodeInto(&input); err != nil {
		ctx.RespAutoError(err)
		return
	}

	if len(input.HostIDs) > common.BKMaxRecordsAtOnce {
		ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrExceedMaxOperationRecordsAtOnce, common.BKMaxRecordsAtOnce))
		return
	}

	txnErr := s.Engine.CoreAPI.CoreService().Txn().AutoRunTxn(ctx.Kit.Ctx, ctx.Kit.Header, func() error {
		ccErr := s.CoreAPI.CoreService().Host().UpdateHostCloudAreaField(ctx.Kit.Ctx, ctx.Kit.Header, input)
		if ccErr != nil {
			blog.ErrorJSON("UpdateHostCloudAreaField failed, core service UpdateHostCloudAreaField failed, input: %s, err: %s, rid: %s", input, ccErr.Error(), rid)
			return ccErr
		}
		return nil
	})

	if txnErr != nil {
		ctx.RespAutoError(txnErr)
		return
	}

	// response success
	ctx.RespEntity(nil)
}

// addPlatSyncTaskIDs add sync task ids to plat info
func (s *Service) addPlatSyncTaskIDs(ctx *rest.Contexts, data *[]mapstr.MapStr) error {
	option := &metadata.SearchCloudOption{
		Page: meta.BasePage{
			Limit: common.BKNoLimit,
		},
		Fields: []string{common.BKCloudSyncTaskID, common.BKCloudSyncVpcs},
	}
	result, err := s.CoreAPI.CoreService().Cloud().SearchSyncTask(ctx.Kit.Ctx, ctx.Kit.Header, option)
	if err != nil {
		blog.Errorf("addPlatSyncTaskIDs failed, rid:%s, option:%+v, err:%+v", ctx.Kit.Rid, option, err)
		return err
	}
	cloudIDTasks := make(map[int64][]int64)
	for _, task := range result.Info {
		for _, vpc := range task.SyncVpcs {
			cloudIDTasks[vpc.CloudID] = append(cloudIDTasks[vpc.CloudID], task.TaskID)
		}
	}

	for i, area := range *data {
		cloudID, err := area.Int64(common.BKCloudIDField)
		if err != nil {
			blog.ErrorJSON("addPlatSyncTaskIDs failed, Int64 err: %v, area:%#v, rid: %s", err, area, ctx.Kit.Rid)
			return err
		}
		if _, ok := cloudIDTasks[cloudID]; ok {
			(*data)[i]["sync_task_ids"] = cloudIDTasks[cloudID]
		} else {
			(*data)[i]["sync_task_ids"] = []int64{}
		}
	}

	return nil
}

// FindCloudAreaHostCount find host count in every cloudarea
func (s *Service) FindCloudAreaHostCount(ctx *rest.Contexts) {
	rid := ctx.Kit.Rid
	input := new(metadata.CloudAreaHostCount)
	if err := ctx.DecodeInto(input); nil != err {
		ctx.RespAutoError(err)
		return
	}

	rawErr := input.Validate()
	if rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	res, err := s.CoreAPI.CoreService().Host().FindCloudAreaHostCount(ctx.Kit.Ctx, ctx.Kit.Header, *input)
	if nil != err {
		blog.Errorf("FindCloudAreaHostCount http do error: %v input:%#v,rid:%s", err, *input, rid)
		ctx.RespAutoError(ctx.Kit.CCError.CCError(common.CCErrCommHTTPDoRequestFailed))
		return
	}
	if false == res.Result {
		blog.Errorf("FindCloudAreaHostCount http reply error.  input:%#v, err code:%d, err msg:%s, rid:%s", *input, res.Code, res.ErrMsg, rid)
		ctx.RespAutoError(res.CCError())
		return
	}

	ctx.RespEntity(res.Data)
}

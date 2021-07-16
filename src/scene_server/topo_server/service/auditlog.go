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
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/http/rest"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/querybuilder"
)

// SearchAuditDict returns all audit types with their name and actions for front-end display
func (s *Service) SearchAuditDict(ctx *rest.Contexts) {
	ctx.RespEntity(metadata.GetAuditDict())
}

// SearchAuditList search audit log list, only contains information for front-end table display
func (s *Service) SearchAuditList(ctx *rest.Contexts) {
	query := metadata.AuditQueryInput{}
	if err := ctx.DecodeInto(&query); nil != err {
		ctx.RespAutoError(err)
		return
	}

	if rawErr := query.Validate(); rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	// the front-end table display fields
	fields := []string{common.BKFieldID, common.BKUser, common.BKResourceTypeField, common.BKActionField,
		common.BKOperationTimeField, common.BKAppIDField, common.BKResourceIDField, common.BKResourceNameField}

	cond := mapstr.MapStr{}
	condition := query.Condition
	if err := condition.Validate(); err != nil {
		blog.Errorf("condition, user and resource_name cannot exist at the same time")
		ctx.RespAutoError(err)
		return
	}

	// parse front-end condition to db search cond
	for _, item := range condition.Condition {
		if item.Operator != querybuilder.OperatorIn && item.Operator != querybuilder.OperatorNotIn {
			if !(item.Field == common.BKResourceNameField && item.Operator == querybuilder.OperatorContains) {
				blog.Errorf("operator invalid, %s wrong, only can be in or not_in (resource_name can use contains)",
					item.Operator)
				ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid,
					"operator only can be in or not_in (resource_name can use contains)"))
				return
			}
		}
		condField, key, err := item.ToMgo()
		if err != nil {
			blog.Errorf("condition invalid, %s wrong, err: %s", key, err.Error())
			ctx.RespAutoError(err)
			return
		}
		cond.Merge(condField)
	}

	if condition.User != "" {
		cond[common.BKUser] = condition.User
	}

	if condition.ResourceName != "" {
		if condition.FuzzyQuery {
			cond[common.BKResourceNameField] = map[string]interface{}{
				common.BKDBLIKE:    condition.ResourceName,
				common.BKDBOPTIONS: "i",
			}
		} else {
			cond[common.BKResourceNameField] = condition.ResourceName
		}
	}

	if condition.ResourceType != "" {
		cond[common.BKResourceTypeField] = condition.ResourceType
	}

	if condition.OperateFrom != "" {
		cond[common.BKOperateFromField] = condition.OperateFrom
	}

	if len(condition.Action) > 0 {
		cond[common.BKActionField] = map[string]interface{}{
			common.BKDBIN: condition.Action,
		}
	}

	if condition.BizID != 0 {
		cond[common.BKAppIDField] = condition.BizID
	}

	if condition.ResourceID != nil {
		cond[common.BKResourceIDField] = condition.ResourceID
	}

	if condition.ObjID != "" {
		switch condition.ResourceType {
		case metadata.ModelInstanceRes:
			cond[common.BKOperationDetailField+"."+common.BKObjIDField] = condition.ObjID
		case metadata.InstanceAssociationRes:
			cond[common.BKOperationDetailField+"."+"src_obj_id"] = condition.ObjID
		default:
			blog.Errorf("unsupported resource type %s when query with object id", condition.ResourceType)
			ctx.RespAutoError(ctx.Kit.CCError.CCErrorf(common.CCErrCommParamsInvalid, common.BKResourceTypeField))
			return
		}
	}

	// parse operation start time and end time from string to time condition
	timeCond, err := parseOperationTimeCondition(ctx.Kit, condition.OperationTime)
	if err != nil {
		ctx.RespAutoError(err)
		return
	}

	if len(timeCond) != 0 {
		cond[common.BKOperationTimeField] = timeCond
	}

	// parse audit type condition by category and audit type condition
	auditTypeCond, notMatch := parseAuditTypeCondition(ctx.Kit, condition)
	if notMatch {
		ctx.RespEntity(map[string]interface{}{"count": 0, "info": []interface{}{}})
		return
	}

	if auditTypeCond != nil {
		cond[common.BKAuditTypeField] = auditTypeCond
	}

	auditQuery := metadata.QueryCondition{
		Condition: cond,
		Fields:    fields,
		Page:      query.Page,
	}
	blog.V(5).Infof("AuditQuery, AuditOperation auditQuery: %+v, rid: %s", auditQuery, ctx.Kit.Rid)

	ctx.SetReadPreference(common.SecondaryPreferredMode)
	count, list, err := s.Logics.AuditOperation().SearchAuditList(ctx.Kit, auditQuery)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}

	ctx.RespEntityWithCount(count, list)
}

// SearchAuditDetail search audit log detail by id
func (s *Service) SearchAuditDetail(ctx *rest.Contexts) {
	query := metadata.AuditDetailQueryInput{}
	if err := ctx.DecodeInto(&query); nil != err {
		ctx.RespAutoError(err)
		return
	}

	if rawErr := query.Validate(); rawErr.ErrCode != 0 {
		ctx.RespAutoError(rawErr.ToCCError(ctx.Kit.CCError))
		return
	}

	cond := make(map[string]interface{})
	cond[common.BKFieldID] = map[string]interface{}{
		common.BKDBIN: query.IDs,
	}

	auditDetailQuery := metadata.QueryCondition{
		Condition: cond,
	}
	blog.V(5).Infof("AuditDetailQuery, AuditOperation auditDetailQuery: %+v, rid: %s", auditDetailQuery, ctx.Kit.Rid)

	ctx.SetReadPreference(common.SecondaryPreferredMode)
	list, err := s.Logics.AuditOperation().SearchAuditDetail(ctx.Kit, auditDetailQuery)
	if nil != err {
		ctx.RespAutoError(err)
		return
	}
	ctx.RespEntity(list)
}

func parseOperationTimeCondition(kit *rest.Kit, operationTime metadata.OperationTimeCondition) (map[string]interface{}, error) {
	timeCond := make(map[string]interface{})

	if len(operationTime.Start) != 0 {
		timeCond[common.BKDBGTE] = operationTime.Start
	}

	if len(operationTime.End) != 0 {
		timeCond[common.BKDBLTE] = operationTime.End
	}

	return timeCond, nil
}

func parseAuditTypeCondition(kit *rest.Kit, condition metadata.AuditQueryCondition) (interface{}, bool) {
	if condition.Category != "" {
		auditTypes := metadata.GetAuditTypesByCategory(condition.Category)
		if condition.AuditType != "" {
			flag := false
			if condition.AuditType != metadata.HostType || condition.Category != "resource" {
				for _, audit := range auditTypes {
					if condition.AuditType == audit {
						flag = true
						break
					}
				}

				// audit type and category not match, result is empty
				if !flag {
					return nil, true
				}
			}
		} else {
			return map[string]interface{}{
				common.BKDBIN: auditTypes,
			}, false
		}
	}

	if condition.AuditType != "" {
		return condition.AuditType, false
	}

	return nil, false
}
